package runtime

import (
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/tools"
)

type Event interface{ isEvent() }

type ToolCallEvent struct {
	ToolCall tools.ToolCall
}

func (e *ToolCallEvent) isEvent() {}

type ToolCallResponseEvent struct {
	ToolCall tools.ToolCall
	Response string
}

func (e *ToolCallResponseEvent) isEvent() {}

type AgentMessageEvent struct {
	Message chat.ChatCompletionMessage
}

func (e *AgentMessageEvent) isEvent() {}

type AgentChoiceEvent struct {
	Choice chat.ChatCompletionStreamChoice
}

func (e *AgentChoiceEvent) isEvent() {}

type ErrorEvent struct {
	Error error
}

func (e *ErrorEvent) isEvent() {}
