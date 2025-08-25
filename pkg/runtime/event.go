package runtime

import (
	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
)

type Event interface{ isEvent() }

type ToolCallEvent struct {
	Type     string         `json:"type"`
	ToolCall tools.ToolCall `json:"tool_call"`
}

func ToolCall(toolCall tools.ToolCall) Event {
	return &ToolCallEvent{
		Type:     "tool_call",
		ToolCall: toolCall,
	}
}

func (e *ToolCallEvent) isEvent() {}

type ToolCallConfirmationEvent struct {
	Type      string         `json:"type"`
	ToolCall  tools.ToolCall `json:"tool_call"`
	AgentName string         `json:"agent_name"`
}

func ToolCallConfirmation(toolCall tools.ToolCall, agentName string) Event {
	return &ToolCallConfirmationEvent{
		Type:      "tool_call_confirmation",
		ToolCall:  toolCall,
		AgentName: agentName,
	}
}
func (e *ToolCallConfirmationEvent) isEvent() {}

type ToolCallResponseEvent struct {
	Type     string         `json:"type"`
	ToolCall tools.ToolCall `json:"tool_call"`
	Response string         `json:"response"`
}

func ToolCallResponse(toolCall tools.ToolCall, response string) Event {
	return &ToolCallResponseEvent{
		Type:     "tool_call_response",
		ToolCall: toolCall,
		Response: response,
	}
}
func (e *ToolCallResponseEvent) isEvent() {}

type AgentChoiceEvent struct {
	Type   string                   `json:"type"`
	Agent  string                   `json:"agent"`
	Choice chat.MessageStreamChoice `json:"choice"`
}

func AgentChoice(agent string, choice chat.MessageStreamChoice) Event { //nolint:gocritic
	return &AgentChoiceEvent{
		Type:   "agent_choice",
		Agent:  agent,
		Choice: choice,
	}
}
func (e *AgentChoiceEvent) isEvent() {}

type ErrorEvent struct {
	Type  string `json:"type"`
	Error error  `json:"error"`
}

func Error(err error) Event {
	return &ErrorEvent{
		Type:  "error",
		Error: err,
	}
}
func (e *ErrorEvent) isEvent() {}

type TokenUsageEvent struct {
	Type  string      `json:"type"`
	Usage *chat.Usage `json:"usage"`
}

func TokenUsage(inputTokens, outputTokens int) Event {
	return &TokenUsageEvent{
		Type: "token_usage",
		Usage: &chat.Usage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	}
}
func (e *TokenUsageEvent) isEvent() {}

type SessionTitleEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Title     string `json:"title"`
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
}

func SessionCompaction(sessionID, status string) Event {
	return &SessionCompactionEvent{
		Type:      "session_compaction",
		SessionID: sessionID,
		Status:    status,
	}
}
func (e *SessionCompactionEvent) isEvent() {}
