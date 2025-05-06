package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rumpl/cagent/config"
	"github.com/rumpl/cagent/openai"
	goOpenAI "github.com/sashabaranov/go-openai"
)

// ToolCall represents an OpenAI tool call message
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ToolHandler is a function type for handling tool calls
type ToolHandler func(ctx context.Context, toolCall ToolCall) (string, error)

// Runtime manages the execution of agents
type Runtime struct {
	client     *openai.Client
	messages   []goOpenAI.ChatCompletionMessage
	tools      []goOpenAI.Tool
	toolMap    map[string]ToolHandler
	parentPath string
	subAgents  map[string]*Agent
	cfg        *config.Config
	agent      *Agent
}

// NewRuntime creates a new runtime for an agent
func NewRuntime(cfg *config.Config, agent *Agent, parentPath string) (*Runtime, error) {
	client, err := openai.NewClientFromConfig(cfg, agent.GetModel())
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	runtime := &Runtime{
		client:     client,
		messages:   []goOpenAI.ChatCompletionMessage{},
		tools:      agent.GetTools(),
		toolMap:    make(map[string]ToolHandler),
		parentPath: parentPath,
		subAgents:  make(map[string]*Agent),
		cfg:        cfg,
		agent:      agent,
	}

	// Register the default tools
	runtime.registerDefaultTools()

	// Add system message with instructions
	runtime.messages = append(runtime.messages, goOpenAI.ChatCompletionMessage{
		Role:    "system",
		Content: agent.GetInstructions(),
	})

	if agent.HasSubAgents() {
		subAgents := agent.GetSubAgents()
		subAgentsStr := ""
		for _, subAgent := range subAgents {
			subAgentsStr += subAgent + ": " + agent.GetInstructions() + "\n"
		}
		runtime.messages = append(runtime.messages, goOpenAI.ChatCompletionMessage{
			Role:    "system",
			Content: "You are a multi-agent system, make sure to answer the user query in the most helpful way possible. You have access to these sub-agents: " + subAgentsStr + "\n\nCall the tool transfer_to_agent if another agent can better answer the user query",
		})
	}

	return runtime, nil
}

// registerDefaultTools registers the default tool handlers
func (r *Runtime) registerDefaultTools() {
	// Register agent transfer tool
	r.toolMap["transfer_to_agent"] = r.handleAgentTransfer
}

// Run starts the agent's interaction loop
func (r *Runtime) Run(ctx context.Context, messages []goOpenAI.ChatCompletionMessage, initialPrompt string) error {
	reader := bufio.NewReader(os.Stdin)

	r.messages = append(r.messages, messages...)

	for {
		// If no initial prompt, get user input
		if initialPrompt == "" {
			fmt.Print("\n\nYou: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}

			input = strings.TrimSpace(input)
			if input == "exit" {
				return nil
			}

			// Add the user message to the conversation
			r.messages = append(r.messages, goOpenAI.ChatCompletionMessage{
				Role:    "user",
				Content: input,
			})
		} else {
			r.messages = append(r.messages, goOpenAI.ChatCompletionMessage{
				Role:    "user",
				Content: initialPrompt,
			})
			initialPrompt = ""
		}

		// Create a streaming chat completion
		stream, err := r.client.CreateChatCompletionStream(ctx, r.messages, r.tools)
		if err != nil {
			return fmt.Errorf("error creating chat completion: %w", err)
		}
		defer stream.Close()

		// Process the response
		var fullContent strings.Builder
		var toolCalls []ToolCall

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return fmt.Errorf("error receiving from stream: %w", err)
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
					toolCalls = append(toolCalls, ToolCall{})
				}

				// Update tool calls with the delta
				for _, deltaToolCall := range choice.Delta.ToolCalls {
					if deltaToolCall.Index == nil {
						continue
					}

					idx := *deltaToolCall.Index
					if idx >= len(toolCalls) {
						// Expand the slice if needed
						newToolCalls := make([]ToolCall, idx+1)
						copy(newToolCalls, toolCalls)
						toolCalls = newToolCalls
					}

					// Update fields based on what's in the delta
					if deltaToolCall.ID != "" {
						toolCalls[idx].ID = deltaToolCall.ID
					}
					if deltaToolCall.Type != "" {
						// Convert the ToolType to string
						toolCalls[idx].Type = string(deltaToolCall.Type)
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
				fmt.Print(choice.Delta.Content)
				fullContent.WriteString(choice.Delta.Content)
			}
		}

		// Add assistant message to conversation history
		assistantMessage := goOpenAI.ChatCompletionMessage{
			Role:    "assistant",
			Content: fullContent.String(),
		}

		// Note: Since we're having issues with the version of the go-openai library,
		// we're not adding the tool calls directly to the message. Instead, we'll
		// add the assistant message and then process the tool calls separately.
		r.messages = append(r.messages, assistantMessage)

		// Handle tool calls if present
		if len(toolCalls) > 0 {
			for _, toolCall := range toolCalls {
				// Call the appropriate tool handler
				handler, exists := r.toolMap[toolCall.Function.Name]
				if !exists {
					fmt.Printf("Error: Tool '%s' not implemented\n", toolCall.Function.Name)
					continue
				}

				result, err := handler(ctx, toolCall)
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
				r.messages = append(r.messages, toolResponseMsg)
			}
		}
	}
}

// handleAgentTransfer handles the transfer_to_agent tool call
func (r *Runtime) handleAgentTransfer(ctx context.Context, toolCall ToolCall) (string, error) {
	var params struct {
		Agent string `json:"agent"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	fmt.Println("Transferring to sub-agent", params.Agent, "initial prompt:", r.messages[len(r.messages)-2].Content)
	// Check if the agent is in the list of subAgents
	if !r.agent.IsSubAgent(params.Agent) {
		return "", fmt.Errorf("agent %s is not a valid sub-agent", params.Agent)
	}

	// Create sub-agent if it doesn't exist
	subAgent, exists := r.subAgents[params.Agent]
	if !exists {
		var err error
		subAgent, err = NewAgent(r.cfg, params.Agent, r.parentPath)
		if err != nil {
			return "", fmt.Errorf("failed to create sub-agent %s: %w", params.Agent, err)
		}
		r.subAgents[params.Agent] = subAgent
	}

	// Create a new runtime for the sub-agent
	subRuntime, err := NewRuntime(r.cfg, subAgent, r.parentPath)
	if err != nil {
		return "", fmt.Errorf("failed to create runtime for sub-agent %s: %w", params.Agent, err)
	}

	// Run the sub-agent with the initial prompt
	subRuntime.Run(ctx, []goOpenAI.ChatCompletionMessage{}, r.messages[len(r.messages)-2].Content)

	return "", nil
}
