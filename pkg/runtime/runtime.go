package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	cagentopenai "github.com/rumpl/cagent/pkg/openai"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/rumpl/cagent/pkg/tools"
	"github.com/sashabaranov/go-openai"
)

// ToolHandler is a function type for handling tool calls
type ToolHandler func(ctx context.Context, a *agent.Agent, sess *session.Session, toolCall tools.ToolCall, events chan Event) (string, error)

// Runtime manages the execution of agents
type Runtime struct {
	logger       *slog.Logger
	toolMap      map[string]ToolHandler
	agents       map[string]*agent.Agent
	cfg          *config.Config
	currentAgent string
}

// New creates a new runtime for an agent
func New(cfg *config.Config, logger *slog.Logger, agents map[string]*agent.Agent, agentName string) (*Runtime, error) {
	runtime := &Runtime{
		toolMap:      make(map[string]ToolHandler),
		agents:       agents,
		cfg:          cfg,
		logger:       logger,
		currentAgent: agentName,
	}

	return runtime, nil
}

// registerDefaultTools registers the default tool handlers
func (r *Runtime) registerDefaultTools() {
	r.toolMap["transfer_to_agent"] = r.handleAgentTransfer
}

func (r *Runtime) CurrentAgent() *agent.Agent {
	return r.agents[r.currentAgent]
}

// Run starts the agent's interaction loop
func (r *Runtime) RunStream(ctx context.Context, sess *session.Session) <-chan Event {
	events := make(chan Event)

	go func() {
		defer close(events)

		a := r.agents[r.currentAgent]

		// TODO: Do not use openai's client directly, use a factory of some kind
		client, err := cagentopenai.NewClientFromConfig(r.cfg, a.Model())
		if err != nil {
			events <- &ErrorEvent{Error: fmt.Errorf("creating client: %w", err)}
			return
		}

		r.registerDefaultTools()

		for {
			messages := sess.GetMessages(a)

			stream, err := client.CreateChatCompletionStream(ctx, messages, a.Tools())
			if err != nil {
				events <- &ErrorEvent{Error: fmt.Errorf("creating chat completion: %w", err)}
				return
			}
			defer stream.Close()

			var fullContent strings.Builder
			var toolCalls []tools.ToolCall

			for {
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					events <- &ErrorEvent{Error: fmt.Errorf("error receiving from stream: %w", err)}
					return
				}

				choice := response.Choices[0]
				if choice.FinishReason == openai.FinishReasonStop {
					return
				}

				// Handle tool calls
				if choice.Delta.ToolCalls != nil && len(choice.Delta.ToolCalls) > 0 {
					for len(toolCalls) < len(choice.Delta.ToolCalls) {
						toolCalls = append(toolCalls, tools.ToolCall{})
					}

					// Update tool calls with the delta
					for _, deltaToolCall := range choice.Delta.ToolCalls {
						if deltaToolCall.Index == nil {
							continue
						}

						idx := *deltaToolCall.Index
						if idx >= len(toolCalls) {
							// Expand the slice if needed
							newToolCalls := make([]tools.ToolCall, idx+1)
							copy(newToolCalls, toolCalls)
							toolCalls = newToolCalls
						}

						// Update fields based on what's in the delta
						if deltaToolCall.ID != "" {
							toolCalls[idx].ID = deltaToolCall.ID
						}
						if deltaToolCall.Type != "" {
							// Convert the ToolType to string
							toolCalls[idx].Type = tools.ToolType(deltaToolCall.Type)
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
					events <- &AgentChoiceEvent{
						Choice: choice,
					}
					fullContent.WriteString(choice.Delta.Content)
				}
			}

			// Add assistant message to conversation history
			assistantMessage := chat.ChatCompletionMessage{
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
				agentTools := a.Tools()
			outer:
				for _, toolCall := range toolCalls {
					handler, exists := r.toolMap[toolCall.Function.Name]
					if exists {

						events <- &ToolCallEvent{
							ToolCall: toolCall,
						}

						_, err := handler(ctx, a, sess, toolCall, events)
						if err != nil {
							r.logger.Error("Error executing tool", "tool", toolCall.Function.Name, "error", err)
						}

						continue
					}

					for _, tool := range agentTools {
						if tool.Function.Name == toolCall.Function.Name {
							exists = true

							events <- &ToolCallEvent{
								ToolCall: toolCall,
							}

							res, err := tool.Handler.CallTool(ctx, toolCall)
							if err != nil {
								r.logger.Error("Error calling tool", "tool", toolCall.Function.Name, "error", err)
								break outer
							}

							toolResponseMsg := chat.ChatCompletionMessage{
								Role:       "tool",
								Content:    res.Output,
								ToolCallID: toolCall.ID,
							}
							sess.Messages = append(sess.Messages, session.AgentMessage{
								Agent:   a,
								Message: toolResponseMsg,
							})
							break
						}
					}
				}
			}
		}
	}()

	return events
}

// Run starts the agent's interaction loop
func (r *Runtime) Run(ctx context.Context, sess *session.Session) ([]session.AgentMessage, error) {
	eventsChan := r.RunStream(ctx, sess)

	for event := range eventsChan {
		if errEvent, ok := event.(*ErrorEvent); ok {
			return nil, errEvent.Error
		}
	}

	return sess.Messages, nil
}

// handleAgentTransfer handles the transfer_to_agent tool call
func (r *Runtime) handleAgentTransfer(ctx context.Context, a *agent.Agent, sess *session.Session, toolCall tools.ToolCall, evts chan Event) (string, error) {
	var params struct {
		Agent string `json:"agent"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	r.logger.Debug("Transferring to sub-agent", "agent", params.Agent)

	if !a.IsSubAgent(params.Agent) && !a.IsParent(params.Agent) {
		return "", fmt.Errorf("agent %s is not a valid sub-agent", params.Agent)
	}

	toolResponseMsg := chat.ChatCompletionMessage{
		Role:       "tool",
		Content:    "{}",
		ToolCallID: toolCall.ID,
	}
	sess.Messages = append(sess.Messages, session.AgentMessage{
		Agent:   a,
		Message: toolResponseMsg,
	})

	r.currentAgent = params.Agent

	for event := range r.RunStream(ctx, sess) {
		evts <- event
		if errEvent, ok := event.(*ErrorEvent); ok {
			return "", errEvent.Error
		}
	}

	return "{}", nil
}
