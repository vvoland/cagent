package runtime

import (
	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
)

type Event interface {
	isEvent()
	GetAgentName() string
}

// AgentContext carries optional agent attribution for an event.
type AgentContext struct {
	AgentName string `json:"agent_name,omitempty"`
}

// GetAgentName returns the agent name for events embedding AgentContext.
func (a AgentContext) GetAgentName() string { return a.AgentName }

type ToolCallEvent struct {
	Type     string         `json:"type"`
	ToolCall tools.ToolCall `json:"tool_call"`
	AgentContext
}

func ToolCall(toolCall tools.ToolCall, agentName string) Event {
	return &ToolCallEvent{
		Type:         "tool_call",
		ToolCall:     toolCall,
		AgentContext: AgentContext{AgentName: agentName},
	}
}

func (e *ToolCallEvent) isEvent() {}

type ToolCallConfirmationEvent struct {
	Type     string         `json:"type"`
	ToolCall tools.ToolCall `json:"tool_call"`
	AgentContext
}

func ToolCallConfirmation(toolCall tools.ToolCall, agentName string) Event {
	return &ToolCallConfirmationEvent{
		Type:         "tool_call_confirmation",
		ToolCall:     toolCall,
		AgentContext: AgentContext{AgentName: agentName},
	}
}
func (e *ToolCallConfirmationEvent) isEvent() {}

type ToolCallResponseEvent struct {
	Type     string         `json:"type"`
	ToolCall tools.ToolCall `json:"tool_call"`
	Response string         `json:"response"`
	AgentContext
}

func ToolCallResponse(toolCall tools.ToolCall, response, agentName string) Event {
	return &ToolCallResponseEvent{
		Type:         "tool_call_response",
		ToolCall:     toolCall,
		Response:     response,
		AgentContext: AgentContext{AgentName: agentName},
	}
}
func (e *ToolCallResponseEvent) isEvent() {}

type AgentChoiceEvent struct {
	Type   string                   `json:"type"`
	Choice chat.MessageStreamChoice `json:"choice"`
	AgentContext
}

func AgentChoice(agentName string, choice chat.MessageStreamChoice) Event { //nolint:gocritic
	return &AgentChoiceEvent{
		Type:         "agent_choice",
		Choice:       choice,
		AgentContext: AgentContext{AgentName: agentName},
	}
}
func (e *AgentChoiceEvent) isEvent() {}

type ErrorEvent struct {
	Type  string `json:"type"`
	Error string `json:"error"`
	AgentContext
}

func Error(msg string) Event {
	return &ErrorEvent{
		Type:  "error",
		Error: msg,
	}
}
func (e *ErrorEvent) isEvent() {}

type TokenUsageEvent struct {
	Type  string `json:"type"`
	Usage *Usage `json:"usage"`
	AgentContext
}

type Usage struct {
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	ContextLength int     `json:"context_length"`
	ContextLimit  int     `json:"context_limit"`
	Cost          float64 `json:"cost"`
}

func TokenUsage(inputTokens, outputTokens, contextLength, contextLimit int, cost float64) Event {
	return &TokenUsageEvent{
		Type: "token_usage",
		Usage: &Usage{
			ContextLength: contextLength,
			ContextLimit:  contextLimit,
			InputTokens:   inputTokens,
			OutputTokens:  outputTokens,
			Cost:          cost,
		},
	}
}
func (e *TokenUsageEvent) isEvent() {}

type SessionTitleEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Title     string `json:"title"`
	AgentContext
}

func SessionTitle(sessionID, title string) Event {
	return &SessionTitleEvent{
		Type:      "session_title",
		SessionID: sessionID,
		Title:     title,
	}
}
func (e *SessionTitleEvent) isEvent() {}

type SessionSummaryEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Summary   string `json:"summary"`
	AgentContext
}

func SessionSummary(sessionID, summary string) Event {
	return &SessionSummaryEvent{
		Type:      "session_summary",
		SessionID: sessionID,
		Summary:   summary,
	}
}
func (e *SessionSummaryEvent) isEvent() {}

type SessionCompactionEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	AgentContext
}

func SessionCompaction(sessionID, status string) Event {
	return &SessionCompactionEvent{
		Type:      "session_compaction",
		SessionID: sessionID,
		Status:    status,
	}
}
func (e *SessionCompactionEvent) isEvent() {}
