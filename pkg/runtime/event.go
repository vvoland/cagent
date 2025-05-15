package runtime

import (
	"github.com/sashabaranov/go-openai"
)

type Event interface{ isEvent() }

type ToolCallEvent struct {
	ToolCall openai.ToolCall
}

func (e *ToolCallEvent) isEvent() {}

type ToolCallResponseEvent struct {
	ToolCall openai.ToolCall
	Response string
}

func (e *ToolCallResponseEvent) isEvent() {}

type AgentMessageEvent struct {
	Message openai.ChatCompletionMessage
}

func (e *AgentMessageEvent) isEvent() {}

type AgentChoiceEvent struct {
	Choice openai.ChatCompletionStreamChoice
}

func (e *AgentChoiceEvent) isEvent() {}

type ErrorEvent struct {
	Error error
}

func (e *ErrorEvent) isEvent() {}
