package dmr

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/sashabaranov/go-openai"

	"github.com/docker/cagent/pkg/chat"
	latest "github.com/docker/cagent/pkg/config/v1"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/tools"
)

// Client represents an DMR client wrapper
// It implements the provider.Provider interface
type Client struct {
	client  *openai.Client
	config  *latest.ModelConfig
	baseURL string
	logger  *slog.Logger
}

// NewClient creates a new DMR client from the provided configuration
func NewClient(_ context.Context, cfg *latest.ModelConfig, logger *slog.Logger, opts ...options.Opt) (*Client, error) {
	if cfg == nil {
		logger.Error("DMR client creation failed", "error", "model configuration is required")
		return nil, errors.New("model configuration is required")
	}

	if cfg.Provider != "dmr" {
		logger.Error("DMR client creation failed", "error", "model type must be 'dmr'", "actual_type", cfg.Provider)
		return nil, errors.New("model type must be 'dmr'")
	}

	var globalOptions options.ModelOptions
	for _, opt := range opts {
		opt(&globalOptions)
	}

	// Set default base_url for DMR models if not provided
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:12434/engines/llama.cpp/v1"
		logger.Debug("Using default DMR base_url", "base_url", baseURL)
	}

	logger.Debug("Creating DMR client config", "base_url", baseURL)
	clientConfig := openai.DefaultConfig("")
	clientConfig.BaseURL = baseURL

	client := openai.NewClientWithConfig(clientConfig)
	logger.Debug("DMR client created successfully", "model", cfg.Model, "base_url", baseURL)

	return &Client{
		client:  client,
		config:  cfg,
		baseURL: baseURL,
		logger:  logger,
	}, nil
}

func convertMultiContent(multiContent []chat.MessagePart) []openai.ChatMessagePart {
	openaiMultiContent := make([]openai.ChatMessagePart, len(multiContent))
	for i, part := range multiContent {
		openaiPart := openai.ChatMessagePart{
			Type: openai.ChatMessagePartType(part.Type),
			Text: part.Text,
		}

		// Handle image URL conversion
		if part.Type == chat.MessagePartTypeImageURL && part.ImageURL != nil {
			openaiPart.ImageURL = &openai.ChatMessageImageURL{
				URL:    part.ImageURL.URL,
				Detail: openai.ImageURLDetail(part.ImageURL.Detail),
			}
		}

		openaiMultiContent[i] = openaiPart
	}
	return openaiMultiContent
}

func convertMessages(messages []chat.Message) []openai.ChatCompletionMessage {
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i := range messages {
		msg := &messages[i]
		role := msg.Role
		if role == chat.MessageRoleSystem {
			role = chat.MessageRoleUser
		}
		openaiMessage := openai.ChatCompletionMessage{
			Role: string(role),
			Name: msg.Name,
		}

		if len(msg.MultiContent) == 0 {
			openaiMessage.Content = msg.Content
		} else {
			openaiMessage.MultiContent = convertMultiContent(msg.MultiContent)
		}

		if msg.FunctionCall != nil {
			openaiMessage.FunctionCall = &openai.FunctionCall{
				Name:      msg.FunctionCall.Name,
				Arguments: msg.FunctionCall.Arguments,
			}
		}

		if len(msg.ToolCalls) > 0 {
			openaiMessage.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for j, toolCall := range msg.ToolCalls {
				openaiMessage.ToolCalls[j] = openai.ToolCall{
					ID:   toolCall.ID,
					Type: openai.ToolType(toolCall.Type),
					Function: openai.FunctionCall{
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					},
				}
			}
		}

		if msg.ToolCallID != "" {
			openaiMessage.ToolCallID = msg.ToolCallID
		}

		openaiMessages[i] = openaiMessage
	}

	var mergedMessages []openai.ChatCompletionMessage

	for i := 0; i < len(openaiMessages); i++ {
		currentMsg := openaiMessages[i]

		if currentMsg.Role == string(chat.MessageRoleSystem) || currentMsg.Role == string(chat.MessageRoleUser) {
			var mergedContent string
			var mergedMultiContent []openai.ChatMessagePart
			j := i

			for j < len(openaiMessages) && openaiMessages[j].Role == currentMsg.Role {
				msgToMerge := openaiMessages[j]

				if len(msgToMerge.MultiContent) == 0 {
					if mergedContent != "" {
						mergedContent += "\n"
					}
					mergedContent += msgToMerge.Content
				} else {
					mergedMultiContent = append(mergedMultiContent, msgToMerge.MultiContent...)
				}
				j++
			}

			mergedMessage := openai.ChatCompletionMessage{
				Role: currentMsg.Role,
			}

			if len(mergedMultiContent) == 0 {
				mergedMessage.Content = mergedContent
			} else {
				mergedMessage.MultiContent = mergedMultiContent
			}

			mergedMessages = append(mergedMessages, mergedMessage)

			i = j - 1
		} else {
			mergedMessages = append(mergedMessages, currentMsg)
		}
	}

	return mergedMessages
}

// CreateChatCompletionStream creates a streaming chat completion request
// It returns a stream that can be iterated over to get completion chunks
func (c *Client) CreateChatCompletionStream(
	ctx context.Context,
	messages []chat.Message,
	requestTools []tools.Tool,
) (chat.MessageStream, error) {
	c.logger.Debug("Creating DMR chat completion stream",
		"model", c.config.Model,
		"message_count", len(messages),
		"tool_count", len(requestTools),
		"base_url", c.baseURL)

	if len(messages) == 0 {
		c.logger.Error("DMR stream creation failed", "error", "at least one message is required")
		return nil, errors.New("at least one message is required")
	}

	parallelToolCalls := true
	if c.config.ParallelToolCalls != nil {
		parallelToolCalls = *c.config.ParallelToolCalls
	}

	request := openai.ChatCompletionRequest{
		Model:             c.config.Model,
		Messages:          convertMessages(messages),
		Temperature:       float32(c.config.Temperature),
		TopP:              float32(c.config.TopP),
		FrequencyPenalty:  float32(c.config.FrequencyPenalty),
		PresencePenalty:   float32(c.config.PresencePenalty),
		Stream:            true,
		ParallelToolCalls: parallelToolCalls,
	}

	if c.config.MaxTokens > 0 {
		request.MaxTokens = c.config.MaxTokens
		c.logger.Debug("DMR request configured with max tokens", "max_tokens", c.config.MaxTokens)
	}

	if len(requestTools) > 0 {
		c.logger.Debug("Adding tools to DMR request", "tool_count", len(requestTools))
		request.Tools = make([]openai.Tool, len(requestTools))
		for i, tool := range requestTools {
			request.Tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Strict:      tool.Function.Strict,
					Parameters:  tool.Function.Parameters,
				},
			}
			if len(tool.Function.Parameters.Properties) == 0 {
				request.Tools[i].Function.Parameters = json.RawMessage("{}")
			}
			c.logger.Debug("Added tool to DMR request", "tool_name", tool.Function.Name)
		}
	}

	// Log the request in JSON format for debugging
	if requestJSON, err := json.Marshal(request); err == nil {
		c.logger.Debug("DMR chat completion request", "request", string(requestJSON))
	} else {
		c.logger.Error("Failed to marshal DMR request to JSON", "error", err)
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		c.logger.Error("DMR stream creation failed", "error", err, "model", c.config.Model, "base_url", c.baseURL)
		return nil, err
	}

	c.logger.Debug("DMR chat completion stream created successfully", "model", c.config.Model)
	return &StreamAdapter{stream: stream}, nil
}

func (c *Client) CreateChatCompletion(
	ctx context.Context,
	messages []chat.Message,
) (string, error) {
	c.logger.Debug("Creating DMR chat completion", "model", c.config.Model, "message_count", len(messages), "base_url", c.baseURL)

	parallelToolCalls := true
	if c.config.ParallelToolCalls != nil {
		parallelToolCalls = *c.config.ParallelToolCalls
	}

	request := openai.ChatCompletionRequest{
		Model:             c.config.Model,
		Messages:          convertMessages(messages),
		ParallelToolCalls: parallelToolCalls,
	}

	response, err := c.client.CreateChatCompletion(ctx, request)
	if err != nil {
		c.logger.Error("DMR chat completion failed", "error", err, "model", c.config.Model, "base_url", c.baseURL)
		return "", err
	}

	c.logger.Debug("DMR chat completion successful", "model", c.config.Model, "response_length", len(response.Choices[0].Message.Content))
	return response.Choices[0].Message.Content, nil
}

func (c *Client) ID() string {
	return c.config.Provider + "/" + c.config.Model
}
