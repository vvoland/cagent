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
	"github.com/rumpl/cagent/pkg/model/provider"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/rumpl/cagent/pkg/tools"
)

// ToolHandler is a function type for handling tool calls
type ToolHandler func(ctx context.Context, a *agent.Agent, sess *session.Session, toolCall tools.ToolCall, events chan Event) (string, error)

// Runtime manages the execution of agents
type Runtime struct {
	logger          *slog.Logger
	toolMap         map[string]ToolHandler
	agents          map[string]*agent.Agent
	cfg             *config.Config
	currentAgent    string
	providerFactory provider.Factory
}

// New creates a new runtime for an agent
func New(cfg *config.Config, logger *slog.Logger, agents map[string]*agent.Agent, agentName string) (*Runtime, error) {
	runtime := &Runtime{
		toolMap:         make(map[string]ToolHandler),
		agents:          agents,
		cfg:             cfg,
		logger:          logger,
		currentAgent:    agentName,
		providerFactory: provider.NewFactory(),
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

		// Create a provider for the agent's model
		modelProvider, err := r.providerFactory.NewProviderFromConfig(r.cfg, a.Model())
		if err != nil {
			events <- &ErrorEvent{Error: fmt.Errorf("creating model provider: %w", err)}
			return
		}

		r.registerDefaultTools()

		for {
			messages := sess.GetMessages(a)

			agentTools, err := a.Tools()
			if err != nil {
				events <- &ErrorEvent{Error: fmt.Errorf("failed to get tools: %w", err)}
				return
			}

			stream, err := modelProvider.CreateChatCompletionStream(ctx, messages, agentTools)
			if err != nil {
				events <- &ErrorEvent{Error: fmt.Errorf("creating chat completion: %w", err)}
				return
			}

			calls, content, stopped, err := r.handleStream(stream, a, events)
			if err != nil {
				events <- &ErrorEvent{Error: err}
				return
			}

			// Add assistant message to conversation history
			assistantMessage := chat.Message{
				Role:      "assistant",
				Content:   content,
				ToolCalls: calls,
			}

			sess.Messages = append(sess.Messages, session.AgentMessage{
				Agent:   a,
				Message: assistantMessage,
			})

			if stopped {
				break
			}

			// Handle tool calls if present
			if len(calls) > 0 {
				agentTools, err := a.Tools()
				if err != nil {
					events <- &ErrorEvent{Error: fmt.Errorf("failed to get tools: %w", err)}
					return
				}

			outer:
				for _, toolCall := range calls {
					handler, exists := r.toolMap[toolCall.Function.Name]
					if exists {
						events <- &ToolCallEvent{
							ToolCall: toolCall,
						}

						res, err := handler(ctx, a, sess, toolCall, events)
						events <- &ToolCallResponseEvent{
							ToolCall: toolCall,
							Response: res,
						}
						if err != nil {
							r.logger.Error("Error executing tool", "tool", toolCall.Function.Name, "error", err)
						}

						return
					}

					for _, tool := range agentTools {
						if tool.Function.Name != toolCall.Function.Name {
							continue
						}
						events <- &ToolCallEvent{
							ToolCall: toolCall,
						}

						res, err := tool.Handler.CallTool(ctx, toolCall)
						if err != nil {
							r.logger.Error("Error calling tool", "tool", toolCall.Function.Name, "error", err)
							break outer
						}

						events <- &ToolCallResponseEvent{
							ToolCall: toolCall,
							Response: res.Output,
						}

						toolResponseMsg := chat.Message{
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

// handleStream handles the stream processing
func (r *Runtime) handleStream(stream chat.MessageStream, a *agent.Agent, events chan Event) (calls []tools.ToolCall, content string, stopped bool, err error) {
	defer stream.Close()

	var fullContent strings.Builder
	var toolCalls []tools.ToolCall

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", false, fmt.Errorf("error receiving from stream: %w", err)
		}

		choice := response.Choices[0]
		if choice.FinishReason == chat.FinishReasonStop {
			return toolCalls, fullContent.String(), true, nil
		}

		// Handle tool calls
		if len(choice.Delta.ToolCalls) > 0 {
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
					newToolCalls := make([]tools.ToolCall, idx+1)
					copy(newToolCalls, toolCalls)
					toolCalls = newToolCalls
				}

				// Update fields based on what's in the delta
				if deltaToolCall.ID != "" {
					toolCalls[idx].ID = deltaToolCall.ID
				}
				if deltaToolCall.Type != "" {
					toolCalls[idx].Type = deltaToolCall.Type
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
				Agent:  a.Name(),
				Choice: choice,
			}
			fullContent.WriteString(choice.Delta.Content)
		}
	}

	return toolCalls, fullContent.String(), false, nil
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

	toolResponseMsg := chat.Message{
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
