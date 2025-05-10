package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/rumpl/cagent/agent"
	"github.com/rumpl/cagent/config"
	cagentopenai "github.com/rumpl/cagent/openai"
	"github.com/sashabaranov/go-openai"
)

// ToolHandler is a function type for handling tool calls
type ToolHandler func(ctx context.Context, a *agent.Agent, toolCall openai.ToolCall) (string, error)

// Runtime manages the execution of agents
type Runtime struct {
	logger    *slog.Logger
	toolMap   map[string]ToolHandler
	subAgents map[string]*agent.Agent
	cfg       *config.Config
}

// NewRuntime creates a new runtime for an agent
func NewRuntime(cfg *config.Config, logger *slog.Logger) (*Runtime, error) {
	runtime := &Runtime{
		toolMap:   make(map[string]ToolHandler),
		subAgents: make(map[string]*agent.Agent),
		cfg:       cfg,
		logger:    logger,
	}

	return runtime, nil
}

// registerDefaultTools registers the default tool handlers
func (r *Runtime) registerDefaultTools() {
	// Register agent transfer tool
	r.toolMap["transfer_to_agent"] = r.handleAgentTransfer
}

// Run starts the agent's interaction loop
func (r *Runtime) Run(ctx context.Context, a *agent.Agent, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	client, err := cagentopenai.NewClientFromConfig(r.cfg, a.GetModel())
	// r.messages = append(r.messages, messages...)
	// Register the default tools
	r.registerDefaultTools()

	// Add system message with instructions
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    "system",
		Content: a.GetInstructions(),
	})

	if a.HasSubAgents() {
		subAgents := a.GetSubAgents()
		subAgentsStr := ""
		for _, subAgent := range subAgents {
			subAgentsStr += subAgent + ": " + a.GetDescription() + "\n"
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "system",
			Content: "You are a multi-agent system, make sure to answer the user query in the most helpful way possible. You have access to these sub-agents: " + subAgentsStr + "\n\nCall the tool transfer_to_agent if another agent can better answer the user query",
		})
	}

	// Create a streaming chat completion
	stream, err := client.CreateChatCompletionStream(ctx, messages, a.GetTools())
	if err != nil {
		return nil, fmt.Errorf("error creating chat completion: %w", err)
	}
	defer stream.Close()

	// Process the response
	var fullContent strings.Builder
	var toolCalls []openai.ToolCall

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error receiving from stream: %w", err)
		}

		// Process the choice
		choice := response.Choices[0]

		// Handle tool calls in the delta
		if choice.Delta.ToolCalls != nil && len(choice.Delta.ToolCalls) > 0 {
			for len(toolCalls) < len(choice.Delta.ToolCalls) {
				toolCalls = append(toolCalls, openai.ToolCall{})
			}

			// Update tool calls with the delta
			for _, deltaToolCall := range choice.Delta.ToolCalls {
				if deltaToolCall.Index == nil {
					continue
				}

				idx := *deltaToolCall.Index
				if idx >= len(toolCalls) {
					// Expand the slice if needed
					newToolCalls := make([]openai.ToolCall, idx+1)
					copy(newToolCalls, toolCalls)
					toolCalls = newToolCalls
				}

				// Update fields based on what's in the delta
				if deltaToolCall.ID != "" {
					toolCalls[idx].ID = deltaToolCall.ID
				}
				if deltaToolCall.Type != "" {
					// Convert the ToolType to string
					toolCalls[idx].Type = openai.ToolType(deltaToolCall.Type)
				}
				if deltaToolCall.Function.Name != "" {
					toolCalls[idx].Function.Name = deltaToolCall.Function.Name
				}
				if deltaToolCall.Function.Arguments != "" {
					if toolCalls[idx].Function.Arguments == "" {
						toolCalls[idx].Function.Arguments = deltaToolCall.Function.Arguments
					} else {
						toolCalls[idx].Function.Arguments += deltaToolCall.Function.Arguments
					}
				}
			}
			continue
		}

		// Print content delta
		if choice.Delta.Content != "" {
			fullContent.WriteString(choice.Delta.Content)
		}
	}

	// Add assistant message to conversation history
	assistantMessage := openai.ChatCompletionMessage{
		Role:      "assistant",
		Content:   fullContent.String(),
		ToolCalls: toolCalls,
	}

	messages = append(messages, assistantMessage)

	// Handle tool calls if present
	if len(toolCalls) > 0 {
		for _, toolCall := range toolCalls {
			// Call the appropriate tool handler
			handler, exists := r.toolMap[toolCall.Function.Name]
			if !exists {
				r.logger.Error("tool not implemented", "name", toolCall.Function.Name)
				continue
			}

			result, err := handler(ctx, a, toolCall)
			if err != nil {
				r.logger.Error("Error executing tool", "tool", toolCall.Function.Name, "error", err)
				result = fmt.Sprintf("Error: %v", err)
			}

			// Add the tool result to the conversation
			toolResponseMsg := openai.ChatCompletionMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: toolCall.ID,
			}
			messages = append(messages, toolResponseMsg)
		}
	}
	return messages, nil
	// }
}

// handleAgentTransfer handles the transfer_to_agent tool call
func (r *Runtime) handleAgentTransfer(ctx context.Context, a *agent.Agent, toolCall openai.ToolCall) (string, error) {
	var params struct {
		Agent string `json:"agent"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	r.logger.Info("Transferring to sub-agent", "agent", params.Agent)
	// Check if the agent is in the list of subAgents
	if !a.IsSubAgent(params.Agent) {
		return "", fmt.Errorf("agent %s is not a valid sub-agent", params.Agent)
	}

	// Create sub-agent if it doesn't exist
	subAgent, exists := r.subAgents[params.Agent]
	if !exists {
		var err error
		subAgent, err = agent.New(r.cfg, params.Agent)
		if err != nil {
			return "", fmt.Errorf("failed to create sub-agent %s: %w", params.Agent, err)
		}
		r.subAgents[params.Agent] = subAgent
	}

	// Create a new runtime for the sub-agent
	subRuntime, err := NewRuntime(r.cfg, r.logger)
	if err != nil {
		return "", fmt.Errorf("failed to create runtime for sub-agent %s: %w", params.Agent, err)
	}

	// Run the sub-agent with the initial prompt
	subRuntime.Run(ctx, subAgent, []openai.ChatCompletionMessage{})

	return "", nil
}
