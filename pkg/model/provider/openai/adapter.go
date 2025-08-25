package openai

import (
	"github.com/sashabaranov/go-openai"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
)

// StreamAdapter adapts the OpenAI stream to our interface
type StreamAdapter struct {
	stream           *openai.ChatCompletionStream
	lastFinishReason chat.FinishReason
}

// Recv gets the next completion chunk
func (a *StreamAdapter) Recv() (chat.MessageStreamResponse, error) {
	openaiResponse, err := a.stream.Recv()
	if err != nil {
		return chat.MessageStreamResponse{}, err
	}

	// Convert the OpenAI response to our generic format
	response := chat.MessageStreamResponse{
		ID:      openaiResponse.ID,
		Object:  openaiResponse.Object,
		Created: openaiResponse.Created,
		Model:   openaiResponse.Model,
		Choices: make([]chat.MessageStreamChoice, len(openaiResponse.Choices)),
	}

	if openaiResponse.Usage != nil {
		response.Usage = &chat.Usage{
			InputTokens:  openaiResponse.Usage.PromptTokens,
			OutputTokens: openaiResponse.Usage.CompletionTokens,
		}
		// Use the tracked finish reason instead of hardcoding stop
		finishReason := a.lastFinishReason
		if finishReason == "" {
			finishReason = chat.FinishReasonStop
		}
		response.Choices = append(response.Choices, chat.MessageStreamChoice{
			FinishReason: finishReason,
		})
	}

	// Convert the choices
	for i := range openaiResponse.Choices {
		choice := &openaiResponse.Choices[i]
		if choice.FinishReason == openai.FinishReasonStop {
			choice.FinishReason = openai.FinishReasonNull
		}

		finishReason := chat.FinishReason(choice.FinishReason)
		// Track the finish reason for when we get usage info
		if finishReason != chat.FinishReasonNull && finishReason != "" {
			a.lastFinishReason = finishReason
		}

		response.Choices[i] = chat.MessageStreamChoice{
			Index:        choice.Index,
			FinishReason: finishReason,
			Delta: chat.MessageDelta{
				Role:    choice.Delta.Role,
				Content: choice.Delta.Content,
			},
		}

		// Convert function call if present
		if choice.Delta.FunctionCall != nil {
			response.Choices[i].Delta.FunctionCall = &tools.FunctionCall{
				Name:      choice.Delta.FunctionCall.Name,
				Arguments: choice.Delta.FunctionCall.Arguments,
			}
		}

		// Convert tool calls if present
		if len(choice.Delta.ToolCalls) > 0 {
			response.Choices[i].Delta.ToolCalls = make([]tools.ToolCall, len(choice.Delta.ToolCalls))
			for j, toolCall := range choice.Delta.ToolCalls {
				response.Choices[i].Delta.ToolCalls[j] = tools.ToolCall{
					ID:   toolCall.ID,
					Type: tools.ToolType(toolCall.Type),
					Function: tools.FunctionCall{
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					},
				}
				// Handle Index field if present
				if toolCall.Index != nil {
					index := *toolCall.Index
					response.Choices[i].Delta.ToolCalls[j].Index = &index
				}
			}
		}
	}

	return response, nil
}

// Close closes the stream
func (a *StreamAdapter) Close() {
	a.stream.Close()
}
