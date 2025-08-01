package anthropic

import (
	"fmt"
	"io"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
)

// StreamAdapter adapts the Anthropic stream to our interface
type StreamAdapter struct {
	stream   *ssestream.Stream[anthropic.MessageStreamEventUnion]
	toolCall bool
	toolIdx  *int
}

// Recv gets the next completion chunk
func (a *StreamAdapter) Recv() (chat.MessageStreamResponse, error) {
	if !a.stream.Next() {
		if a.stream.Err() != nil {
			return chat.MessageStreamResponse{}, a.stream.Err()
		}
		return chat.MessageStreamResponse{}, io.EOF
	}

	event := a.stream.Current()

	response := chat.MessageStreamResponse{
		ID:     event.Message.ID,
		Object: "chat.completion.chunk",
		Model:  string(event.Message.Model),
		Choices: []chat.MessageStreamChoice{
			{
				Index: 0,
				Delta: chat.MessageDelta{
					Role: string(chat.MessageRoleAssistant),
				},
			},
		},
	}

	// Handle different event types
	switch eventVariant := event.AsAny().(type) {
	case anthropic.ContentBlockStartEvent:
		if contentBlock, ok := eventVariant.ContentBlock.AsAny().(anthropic.ToolUseBlock); ok {
			a.toolCall = true
			if a.toolIdx == nil {
				toolIdx := 0
				a.toolIdx = &toolIdx
			} else {
				*a.toolIdx++
			}
			toolCall := tools.ToolCall{
				ID:    contentBlock.ID,
				Type:  "function",
				Index: a.toolIdx,
				Function: tools.FunctionCall{
					Name: contentBlock.Name,
				},
			}
			response.Choices[0].Delta.ToolCalls = []tools.ToolCall{toolCall}
		}
	case anthropic.ContentBlockDeltaEvent:
		switch deltaVariant := eventVariant.Delta.AsAny().(type) {
		case anthropic.TextDelta:
			response.Choices[0].Delta.Content = deltaVariant.Text

		case anthropic.InputJSONDelta:
			inputBytes := deltaVariant.PartialJSON
			toolCall := tools.ToolCall{
				Type:  "function",
				Index: a.toolIdx,
				Function: tools.FunctionCall{
					Arguments: inputBytes,
				},
			}
			response.Choices[0].Delta.ToolCalls = []tools.ToolCall{toolCall}

		default:
			return response, fmt.Errorf("unknown delta type: %T", deltaVariant)
		}
	case anthropic.MessageStopEvent:
		if a.toolCall {
			response.Choices[0].FinishReason = chat.FinishReasonToolCalls
		} else {
			response.Choices[0].FinishReason = chat.FinishReasonStop
		}
	}

	return response, nil
}

// Close closes the stream
func (a *StreamAdapter) Close() {
	if a.stream != nil {
		a.stream.Close()
	}
}
