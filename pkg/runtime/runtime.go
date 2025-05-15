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
	"github.com/rumpl/cagent/pkg/session"
	"github.com/sashabaranov/go-openai"
)

// ToolHandler is a function type for handling tool calls
type ToolHandler func(ctx context.Context, a *agent.Agent, sess *session.Session, toolCall openai.ToolCall) (string, error)

// Runtime manages the execution of agents
type Runtime struct {
	logger  *slog.Logger
	toolMap map[string]ToolHandler
	agents  map[string]*agent.Agent
	cfg     *config.Config
}

// NewRuntime creates a new runtime for an agent
func NewRuntime(cfg *config.Config, logger *slog.Logger, agents map[string]*agent.Agent) (*Runtime, error) {
	runtime := &Runtime{
		toolMap: make(map[string]ToolHandler),
		agents:  agents,
		cfg:     cfg,
		logger:  logger,
	}

	return runtime, nil
}

// registerDefaultTools registers the default tool handlers
func (r *Runtime) registerDefaultTools() {
	r.toolMap["transfer_to_agent"] = r.handleAgentTransfer
}

// Run starts the agent's interaction loop
func (r *Runtime) Run(ctx context.Context, a *agent.Agent, sess *session.Session) ([]session.AgentMessage, error) {
	client, err := cagentopenai.NewClientFromConfig(r.cfg, a.Model())
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	r.registerDefaultTools()

	messages := sess.GetMessages(a)

	stream, err := client.CreateChatCompletionStream(ctx, messages, a.Tools())
	if err != nil {
		return nil, fmt.Errorf("creating chat completion: %w", err)
	}
	defer stream.Close()

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

		choice := response.Choices[0]

		// Handle tool calls
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

	sess.Messages = append(sess.Messages, session.AgentMessage{
		Agent:   a,
		Message: assistantMessage,
	})

	messages = append(messages, assistantMessage)

	// Handle tool calls if present
	if len(toolCalls) > 0 {
		for _, toolCall := range toolCalls {
			handler, exists := r.toolMap[toolCall.Function.Name]
			if !exists {
				r.logger.Error("tool not implemented", "name", toolCall.Function.Name)
				continue
			}

			result, err := handler(ctx, a, sess, toolCall)
			if err != nil {
				r.logger.Error("Error executing tool", "tool", toolCall.Function.Name, "error", err)
				result = fmt.Sprintf("Error: %v", err)
			}

			if toolCall.Function.Name != "transfer_to_agent" {
				toolResponseMsg := openai.ChatCompletionMessage{
					Role:       "tool",
					Content:    result,
					ToolCallID: toolCall.ID,
				}
				sess.Messages = append(sess.Messages, session.AgentMessage{
					Agent:   a,
					Message: toolResponseMsg,
				})
			}
		}
	}

	return sess.Messages, nil
}

// handleAgentTransfer handles the transfer_to_agent tool call
func (r *Runtime) handleAgentTransfer(ctx context.Context, a *agent.Agent, sess *session.Session, toolCall openai.ToolCall) (string, error) {
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
	subAgent, exists := r.agents[params.Agent]
	if !exists {
		return "", fmt.Errorf("sub-agent %s not found", params.Agent)
	}

	toolResponseMsg := openai.ChatCompletionMessage{
		Role:       "tool",
		Content:    "{}",
		ToolCallID: toolCall.ID,
	}
	sess.Messages = append(sess.Messages, session.AgentMessage{
		Agent:   a,
		Message: toolResponseMsg,
	})

	// Run the sub-agent with the initial prompt
	// We don't need the returned response since the messages are already added to the session
	_, err := r.Run(ctx, subAgent, sess)
	if err != nil {
		return "", fmt.Errorf("failed to run sub-agent %s: %w", params.Agent, err)
	}

	return "{}", nil
}
