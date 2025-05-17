package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/tools"
)

// AnthropicStreamAdapter adapts the Anthropic stream to our interface
type AnthropicStreamAdapter struct {
	stream *ssestream.Stream[anthropic.MessageStreamEventUnion]
}

// Recv gets the next completion chunk
func (a *AnthropicStreamAdapter) Recv() (chat.ChatCompletionStreamResponse, error) {
	if !a.stream.Next() {
		if err := a.stream.Err(); err != nil {
			// Convert Anthropic errors to appropriate chat completion errors
			if apiErr, ok := err.(*anthropic.Error); ok {
				return chat.ChatCompletionStreamResponse{}, fmt.Errorf("anthropic API error (status: %d): %s", apiErr.StatusCode, apiErr.Error())
			}
			return chat.ChatCompletionStreamResponse{}, err
		}
		return chat.ChatCompletionStreamResponse{}, nil
	}

	event := a.stream.Current()

	response := chat.ChatCompletionStreamResponse{
		ID:     event.Message.ID,
		Object: "chat.completion.chunk",
		// Created: event.Message.CreatedAt.Unix(),
		Model: string(event.Message.Model),
		Choices: []chat.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: chat.ChatCompletionDelta{
					Role: "assistant",
				},
			},
		},
	}

	// Handle different event types
	switch eventVariant := event.AsAny().(type) {
	case anthropic.ContentBlockDeltaEvent:
		switch deltaVariant := eventVariant.Delta.AsAny().(type) {
		case anthropic.TextDelta:
			response.Choices[0].Delta.Content = deltaVariant.Text
		case anthropic.ToolUseBlock:
			// Convert tool use to the expected format
			toolCall := tools.ToolCall{
				ID:   deltaVariant.ID,
				Type: "function",
				Function: tools.FunctionCall{
					Name:      deltaVariant.Name,
					Arguments: string(deltaVariant.Input),
				},
			}
			response.Choices[0].Delta.ToolCalls = []tools.ToolCall{toolCall}
			response.Choices[0].FinishReason = chat.FinishReasonToolCalls
		}
	case anthropic.ContentBlockStopEvent:
		response.Choices[0].FinishReason = chat.FinishReasonStop
	case anthropic.MessageStopEvent:
		response.Choices[0].FinishReason = chat.FinishReasonStop
		// case anthropic.Error:
		// 	return chat.ChatCompletionStreamResponse{}, errors.New("stream error: " + eventVariant.Message())
	}

	return response, nil
}

// Close closes the stream
func (a *AnthropicStreamAdapter) Close() {
	if a.stream != nil {
		a.stream.Close()
	}
}

// Client represents an Anthropic client wrapper implementing provider.Provider
// It holds the anthropic client and model config
type Client struct {
	client anthropic.Client
	config *config.ModelConfig
}

// NewClient creates a new Anthropic client from the provided configuration
func NewClient(cfg *config.ModelConfig) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("model configuration is required")
	}
	if cfg.Type != "anthropic" {
		return nil, errors.New("model type must be 'anthropic'")
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, errors.New("ANTHROPIC_API_KEY environment variable is required")
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// GetClient returns the underlying anthropic client
func (c *Client) GetClient() *anthropic.Client {
	return &c.client
}

// GetConfig returns the model configuration
func (c *Client) GetConfig() *config.ModelConfig {
	return c.config
}

// CreateChatCompletionStream creates a streaming chat completion request
func (c *Client) CreateChatCompletionStream(
	ctx context.Context,
	messages []chat.ChatCompletionMessage,
	tools []tools.Tool,
) (chat.ChatCompletionStream, error) {
	stream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7Sonnet20250219,
		MaxTokens: 1024,
		Messages:  convertMessages(messages),
		Tools:     convertTools(tools),
	})

	return &AnthropicStreamAdapter{stream: stream}, nil
}

func convertMessages(messages []chat.ChatCompletionMessage) []anthropic.MessageParam {
	anthropicMessages := make([]anthropic.MessageParam, len(messages))
	for i, msg := range messages {
		if msg.Role == "system" {
			anthropicMessages[i] = anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content))
			continue
		}
		if msg.Role == "user" {
			anthropicMessages[i] = anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content))
			continue
		}
		if msg.Role == "assistant" {
			if len(msg.ToolCalls) > 0 {
				toolUseBlocks := make([]anthropic.ContentBlockParamUnion, len(msg.ToolCalls))
				for i, toolCall := range msg.ToolCalls {
					var inpts map[string]any
					json.Unmarshal([]byte(toolCall.Function.Arguments), &inpts)
					toolUseBlocks[i] = anthropic.ContentBlockParamUnion{
						OfRequestToolUseBlock: &anthropic.ToolUseBlockParam{
							ID:    toolCall.ID,
							Input: inpts,
							Name:  toolCall.Function.Name,
						},
					}
				}
				anthropicMessages[i] = anthropic.NewAssistantMessage(toolUseBlocks...)
			} else {
				anthropicMessages[i] = anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content))
			}
			continue
		}
		if msg.Role == "tool" {
			anthropicMessages[i] = anthropic.NewUserMessage(anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false))
			continue
		}
		fmt.Println("unknown message role", msg.Role)
	}
	return anthropicMessages
}

func convertTools(tooles []tools.Tool) []anthropic.ToolUnionParam {
	toolParams := make([]anthropic.ToolParam, len(tooles))

	for i, tool := range tooles {
		toolParams[i] = anthropic.ToolParam{
			Name:        tool.Function.Name,
			Description: anthropic.String(tool.Function.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: tool.Function.Parameters.Properties,
			},
		}
	}
	tools := make([]anthropic.ToolUnionParam, len(toolParams))
	for i, toolParam := range toolParams {
		tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
	}

	return tools
}
