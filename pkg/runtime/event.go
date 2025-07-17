package runtime

import (
	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
)

type Event interface{ isEvent() }

type ToolCallEvent struct {
	ToolCall tools.ToolCall `json:"tool_call"`
}

func (e *ToolCallEvent) isEvent() {}

type ToolCallResponseEvent struct {
	ToolCall tools.ToolCall `json:"tool_call"`
	Response string         `json:"response"`
}

func (e *ToolCallResponseEvent) isEvent() {}

type AgentMessageEvent struct {
	Message chat.Message `json:"message"`
}

func (e *AgentMessageEvent) isEvent() {}

type AgentChoiceEvent struct {
	Agent  string                   `json:"agent"`
	Choice chat.MessageStreamChoice `json:"choice"`
}

func (e *AgentChoiceEvent) isEvent() {}

type ErrorEvent struct {
	Error error `json:"error"`
}

func (e *ErrorEvent) isEvent() {}
