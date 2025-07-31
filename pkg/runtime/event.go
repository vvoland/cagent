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
	Type     string         `json:"type"`
	ToolCall tools.ToolCall `json:"tool_call"`
}

func ToolCallConfirmation(toolCall tools.ToolCall) Event {
	return &ToolCallConfirmationEvent{
		Type:     "tool_call_confirmation",
		ToolCall: toolCall,
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
