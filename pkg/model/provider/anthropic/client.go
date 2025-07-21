package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/config"
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
					Arguments: string(inputBytes),
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

// Client represents an Anthropic client wrapper implementing provider.Provider
// It holds the anthropic client and model config
type Client struct {
	client anthropic.Client
	config *config.ModelConfig
	logger *slog.Logger
}

// NewClient creates a new Anthropic client from the provided configuration
func NewClient(cfg *config.ModelConfig, logger *slog.Logger) (*Client, error) {
	if cfg == nil {
		logger.Error("Anthropic client creation failed", "error", "model configuration is required")
		return nil, errors.New("model configuration is required")
	}
	if cfg.Type != "anthropic" {
		logger.Error("Anthropic client creation failed", "error", "model type must be 'anthropic'", "actual_type", cfg.Type)
		return nil, errors.New("model type must be 'anthropic'")
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		logger.Error("Anthropic client creation failed", "error", "ANTHROPIC_API_KEY environment variable is required")
		return nil, errors.New("ANTHROPIC_API_KEY environment variable is required")
	}

	logger.Debug("Anthropic API key found, creating client")
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	logger.Debug("Anthropic client created successfully", "model", cfg.Model)

	return &Client{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// CreateChatCompletionStream creates a streaming chat completion request
func (c *Client) CreateChatCompletionStream(
	ctx context.Context,
	messages []chat.Message,
	requestTools []tools.Tool,
) (chat.MessageStream, error) {
	c.logger.Debug("Creating Anthropic chat completion stream",
		"model", c.config.Model,
		"message_count", len(messages),
		"tool_count", len(requestTools))

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.config.Model),
		MaxTokens: int64(c.config.MaxTokens),
		Messages:  convertMessages(messages),
		Tools:     convertTools(requestTools),
	}

	if len(requestTools) > 0 {
		c.logger.Debug("Adding tools to Anthropic request", "tool_count", len(requestTools))
	}

	// Log the request details for debugging
	c.logger.Debug("Anthropic chat completion stream request",
		"model", params.Model,
		"max_tokens", params.MaxTokens,
		"message_count", len(params.Messages))

	stream := c.client.Messages.NewStreaming(ctx, params)
	c.logger.Debug("Anthropic chat completion stream created successfully", "model", c.config.Model)

	return &StreamAdapter{stream: stream}, nil
}

func (c *Client) CreateChatCompletion(
	ctx context.Context,
	messages []chat.Message,
) (string, error) {
	c.logger.Debug("Creating Anthropic chat completion", "model", c.config.Model, "message_count", len(messages))

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.config.Model),
		MaxTokens: int64(c.config.MaxTokens),
		Messages:  convertMessages(messages),
	}

	response, err := c.client.Messages.New(ctx, params)
	if err != nil {
		c.logger.Error("Anthropic chat completion failed", "error", err, "model", c.config.Model)
		return "", err
	}

	c.logger.Debug("Anthropic chat completion successful", "model", c.config.Model, "response_length", len(response.Content[0].Text))
	return response.Content[0].Text, nil
}

func convertMessages(messages []chat.Message) []anthropic.MessageParam {
	var anthropicMessages []anthropic.MessageParam

	for i := range messages {
		msg := &messages[i]
		if msg.Role == chat.MessageRoleSystem {
			// Convert system message to user message with system prefix
			systemContent := "System: " + msg.Content
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(strings.TrimSpace(systemContent))))
			continue
		}
		if msg.Role == chat.MessageRoleUser {
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(anthropic.NewTextBlock(strings.TrimSpace(msg.Content))))
			continue
		}
		if msg.Role == chat.MessageRoleAssistant {
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
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(strings.TrimSpace(msg.Content))))
			}
			continue
		}
		if msg.Role == chat.MessageRoleTool {
			toolResult := anthropic.NewToolResultBlock(msg.ToolCallID)
			toolResult.OfToolResult = &anthropic.ToolResultBlockParam{
				ToolUseID: msg.ToolCallID,
				Content: []anthropic.ToolResultBlockParamContentUnion{
					{
						OfText: &anthropic.TextBlockParam{
							Text: strings.TrimSpace(msg.Content),
						},
					},
				},
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(toolResult))
			continue
		}
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
	anthropicTools := make([]anthropic.ToolUnionParam, len(toolParams))
	for i := range toolParams {
		anthropicTools[i] = anthropic.ToolUnionParam{OfTool: &toolParams[i]}
	}

	return anthropicTools
}
