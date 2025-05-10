package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rumpl/cagent/agent"
	"github.com/rumpl/cagent/config"
	"github.com/rumpl/cagent/openai"
	"github.com/rumpl/cagent/pkg/session"
	goOpenAI "github.com/sashabaranov/go-openai"
)

// ToolHandler is a function type for handling tool calls
type ToolHandler func(ctx context.Context, a *agent.Agent, toolCall goOpenAI.ToolCall) (string, error)

// Runtime manages the execution of agents
type Runtime struct {
	toolMap   map[string]ToolHandler
	subAgents map[string]*agent.Agent
	cfg       *config.Config
}

// NewRuntime creates a new runtime for an agent
func NewRuntime(cfg *config.Config) (*Runtime, error) {
	// client, err := openai.NewClientFromConfig(cfg, a.GetModel())
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	// }

	runtime := &Runtime{
		// client:    client,
		toolMap:   make(map[string]ToolHandler),
		subAgents: make(map[string]*agent.Agent),
		cfg:       cfg,
	}

	return runtime, nil
}

// registerDefaultTools registers the default tool handlers
func (r *Runtime) registerDefaultTools() {
	// Register agent transfer tool
	r.toolMap["transfer_to_agent"] = r.handleAgentTransfer
}

func (r *Runtime) RunOnce(ctx context.Context, message string, session *session.Session) (string, error) {
	return "", nil
}

func (r *Runtime) RunOnceWithSession(ctx context.Context, message string, session *session.Session) (string, error) {
	return "", nil
}

// Run starts the agent's interaction loop
func (r *Runtime) Run(ctx context.Context, a *agent.Agent, messages []goOpenAI.ChatCompletionMessage) ([]goOpenAI.ChatCompletionMessage, error) {
	client, err := openai.NewClientFromConfig(r.cfg, a.GetModel())
	// r.messages = append(r.messages, messages...)
	// Register the default tools
	r.registerDefaultTools()

	// Add system message with instructions
	messages = append(messages, goOpenAI.ChatCompletionMessage{
		Role:    "system",
		Content: a.GetInstructions(),
	})

	if a.HasSubAgents() {
		subAgents := a.GetSubAgents()
		subAgentsStr := ""
		for _, subAgent := range subAgents {
			subAgentsStr += subAgent + ": " + a.GetDescription() + "\n"
		}
		messages = append(messages, goOpenAI.ChatCompletionMessage{
			Role:    "system",
			Content: "You are a multi-agent system, make sure to answer the user query in the most helpful way possible. You have access to these sub-agents: " + subAgentsStr + "\n\nCall the tool transfer_to_agent if another agent can better answer the user query",
		})
	}

	// for {
	if messages[len(messages)-1].Role != "tool" {
		// If no initial prompt, get user input
		// if initialPrompt == "" {
		// 	fmt.Print("\n\nYou: ")
		// 	input, err := reader.ReadString('\n')
		// 	if err != nil {
		// 		return fmt.Errorf("error reading input: %w", err)
		// 	}

		// 	input = strings.TrimSpace(input)
		// 	if input == "exit" {
		// 		return nil
		// 	}

		// 	// Add the user message to the conversation
		// 	r.messages = append(r.messages, goOpenAI.ChatCompletionMessage{
		// 		Role:    "user",
		// 		Content: input,
		// 	})
		// } else {
		// 	r.messages = append(r.messages, goOpenAI.ChatCompletionMessage{
		// 		Role:    "user",
		// 		Content: initialPrompt,
		// 	})
		// 	initialPrompt = ""
		// }
	}

	// Create a streaming chat completion
	stream, err := client.CreateChatCompletionStream(ctx, messages, a.GetTools())
	if err != nil {
		return nil, fmt.Errorf("error creating chat completion: %w", err)
	}
	defer stream.Close()

	// Process the response
	var fullContent strings.Builder
	var toolCalls []goOpenAI.ToolCall

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
			// Handle tool calls streaming
			// Note: This is a simplified implementation as the actual structure
			// might vary depending on the go-openai version

			// Ensure we have enough room in our toolCalls slice
			for len(toolCalls) < len(choice.Delta.ToolCalls) {
				toolCalls = append(toolCalls, goOpenAI.ToolCall{})
			}

			// Update tool calls with the delta
			for _, deltaToolCall := range choice.Delta.ToolCalls {
				if deltaToolCall.Index == nil {
					continue
				}

				idx := *deltaToolCall.Index
				if idx >= len(toolCalls) {
					// Expand the slice if needed
					newToolCalls := make([]goOpenAI.ToolCall, idx+1)
					copy(newToolCalls, toolCalls)
					toolCalls = newToolCalls
				}

				// Update fields based on what's in the delta
				if deltaToolCall.ID != "" {
					toolCalls[idx].ID = deltaToolCall.ID
				}
				if deltaToolCall.Type != "" {
					// Convert the ToolType to string
					toolCalls[idx].Type = goOpenAI.ToolType(deltaToolCall.Type)
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
	assistantMessage := goOpenAI.ChatCompletionMessage{
		Role:    "assistant",
		Content: fullContent.String(),
		// Convert our ToolCall slice to the go-openai library's ToolCall type
		ToolCalls: toolCalls,
	}

	// Note: Since we're having issues with the version of the go-openai library,
	// we're not adding the tool calls directly to the message. Instead, we'll
	// add the assistant message and then process the tool calls separately.
	messages = append(messages, assistantMessage)

	// Handle tool calls if present
	if len(toolCalls) > 0 {
		for _, toolCall := range toolCalls {
			// Call the appropriate tool handler
			handler, exists := r.toolMap[toolCall.Function.Name]
			if !exists {
				fmt.Printf("Error: Tool '%s' not implemented\n", toolCall.Function.Name)
				continue
			}

			result, err := handler(ctx, a, toolCall)
			if err != nil {
				fmt.Printf("Error executing tool '%s': %v\n", toolCall.Function.Name, err)
				result = fmt.Sprintf("Error: %v", err)
			}

			// Add the tool result to the conversation
			toolResponseMsg := goOpenAI.ChatCompletionMessage{
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
func (r *Runtime) handleAgentTransfer(ctx context.Context, a *agent.Agent, toolCall goOpenAI.ToolCall) (string, error) {
	var params struct {
		Agent string `json:"agent"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// fmt.Println("Transferring to sub-agent", params.Agent, "initial prompt:", r.messages[len(r.messages)-2].Content)
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
	subRuntime, err := NewRuntime(r.cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create runtime for sub-agent %s: %w", params.Agent, err)
	}

	// Run the sub-agent with the initial prompt
	subRuntime.Run(ctx, subAgent, []goOpenAI.ChatCompletionMessage{})

	return "", nil
}
