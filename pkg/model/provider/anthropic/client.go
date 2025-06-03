package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/tools"
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
					Role: "assistant",
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
					Arguments: string(inputBytes),
				},
			}
			response.Choices[0].Delta.ToolCalls = []tools.ToolCall{toolCall}

		default:
			fmt.Println("Unknown delta type:", deltaVariant)
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
	messages []chat.Message,
	tools []tools.Tool,
) (chat.MessageStream, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7Sonnet20250219,
		MaxTokens: 64000,
		Messages:  convertMessages(messages),
		Tools:     convertTools(tools),
	}

	stream := c.client.Messages.NewStreaming(ctx, params)

	return &StreamAdapter{stream: stream}, nil
}

func convertMessages(messages []chat.Message) []anthropic.MessageParam {
	var anthropicMessages []anthropic.MessageParam

	for i := range messages {
		msg := &messages[i]
		if msg.Role == "system" {
			// Convert system message to user message with system prefix
			systemContent := "System: " + msg.Content
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(systemContent)))
			continue
		}
		if msg.Role == "user" {
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			continue
		}
		if msg.Role == "assistant" {
			if len(msg.ToolCalls) > 0 {
				toolUseBlocks := make([]anthropic.ContentBlockParamUnion, len(msg.ToolCalls))
				for j, toolCall := range msg.ToolCalls {
					var inpts map[string]any
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &inpts); err != nil {
						inpts = map[string]any{}
					}
					toolUseBlocks[j] = anthropic.ContentBlockParamUnion{
						OfToolUse: &anthropic.ToolUseBlockParam{
							ID:    toolCall.ID,
							Input: inpts,
							Name:  toolCall.Function.Name,
						},
					}
				}
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(toolUseBlocks...))
			} else {
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
			continue
		}
		if msg.Role == "tool" {
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false)))
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
	for i := range toolParams {
		tools[i] = anthropic.ToolUnionParam{OfTool: &toolParams[i]}
	}

	return tools
}
