package openai

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/sashabaranov/go-openai"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/desktop"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/tools"
)

// Client represents an OpenAI client wrapper
// It implements the provider.Provider interface
type Client struct {
	client *openai.Client
	config *config.ModelConfig
	logger *slog.Logger
}

// NewClient creates a new OpenAI client from the provided configuration
func NewClient(cfg *config.ModelConfig, env environment.Provider, logger *slog.Logger, opts ...options.Opt) (*Client, error) {
	if cfg == nil {
		logger.Error("OpenAI client creation failed", "error", "model configuration is required")
		return nil, errors.New("model configuration is required")
	}

	if cfg.Type != "openai" {
		logger.Error("OpenAI client creation failed", "error", "model type must be 'openai'", "actual_type", cfg.Type)
		return nil, errors.New("model type must be 'openai'")
	}

	var globalOptions options.ModelOptions
	for _, opt := range opts {
		opt(&globalOptions)
	}

	var openaiConfig openai.ClientConfig
	if gateway := globalOptions.Gateway(); gateway == "" {
		authToken, err := env.Get(context.TODO(), "OPENAI_API_KEY")
		if err != nil || authToken == "" {
			logger.Error("OpenAI client creation failed", "error", "failed to get authentication token", "details", err)
			return nil, errors.New("OPENAI_API_KEY environment variable is required")
		}

		openaiConfig = openai.DefaultConfig(authToken)
	} else {
		authToken := desktop.GetToken(context.TODO())
		if authToken == "" {
			logger.Error("OpenAI client creation failed", "error", "failed to get Docker Desktop's authentication token")
			return nil, errors.New("sorry, you first need to sign in Docker Desktop to use the Docker AI Gateway")
		}

		openaiConfig = openai.DefaultConfig(authToken)
		openaiConfig.BaseURL = gateway + "/v1"
	}

	logger.Debug("OpenAI API key found, creating client")
	client := openai.NewClientWithConfig(openaiConfig)
	logger.Debug("OpenAI client created successfully", "model", cfg.Model)

	return &Client{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

func convertMultiContent(multiContent []chat.MessagePart) []openai.ChatMessagePart {
	openaiMultiContent := make([]openai.ChatMessagePart, len(multiContent))
	for i, part := range multiContent {
		openaiMultiContent[i] = openai.ChatMessagePart{
			Type: openai.ChatMessagePartType(part.Type),
			Text: part.Text,
		}
	}
	return openaiMultiContent
}

// convertMessages converts chat.ChatCompletionMessage to openai.ChatCompletionMessage
func convertMessages(messages []chat.Message) []openai.ChatCompletionMessage {
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i := range messages {
		msg := &messages[i]
		openaiMessage := openai.ChatCompletionMessage{
			Role: string(msg.Role),
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
	return openaiMessages
}

// CreateChatCompletionStream creates a streaming chat completion request
// It returns a stream that can be iterated over to get completion chunks
func (c *Client) CreateChatCompletionStream(
	ctx context.Context,
	messages []chat.Message,
	requestTools []tools.Tool,
) (chat.MessageStream, error) {
	c.logger.Debug("Creating OpenAI chat completion stream",
		"model", c.config.Model,
		"message_count", len(messages),
		"tool_count", len(requestTools))

	if len(messages) == 0 {
		c.logger.Error("OpenAI stream creation failed", "error", "at least one message is required")
		return nil, errors.New("at least one message is required")
	}

	request := openai.ChatCompletionRequest{
		Model:            c.config.Model,
		Messages:         convertMessages(messages),
		Temperature:      float32(c.config.Temperature),
		TopP:             float32(c.config.TopP),
		FrequencyPenalty: float32(c.config.FrequencyPenalty),
		PresencePenalty:  float32(c.config.PresencePenalty),
		Stream:           true,
	}

	if c.config.ParallelToolCalls != nil {
		request.ParallelToolCalls = *c.config.ParallelToolCalls
	}

	if c.config.MaxTokens > 0 {
		request.MaxTokens = c.config.MaxTokens
		c.logger.Debug("OpenAI request configured with max tokens", "max_tokens", c.config.MaxTokens)
	}

	if len(requestTools) > 0 {
		c.logger.Debug("Adding tools to OpenAI request", "tool_count", len(requestTools))
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
			c.logger.Debug("Added tool to OpenAI request", "tool_name", tool.Function.Name)
		}
	}

	// Log the request in JSON format for debugging
	if requestJSON, err := json.Marshal(request); err == nil {
		c.logger.Debug("OpenAI chat completion request", "request", string(requestJSON))
	} else {
		c.logger.Error("Failed to marshal OpenAI request to JSON", "error", err)
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		c.logger.Error("OpenAI stream creation failed", "error", err, "model", c.config.Model)
		return nil, err
	}

	c.logger.Debug("OpenAI chat completion stream created successfully", "model", c.config.Model)
	return &StreamAdapter{stream: stream}, nil
}

func (c *Client) CreateChatCompletion(
	ctx context.Context,
	messages []chat.Message,
) (string, error) {
	c.logger.Debug("Creating OpenAI chat completion", "model", c.config.Model, "message_count", len(messages))

	request := openai.ChatCompletionRequest{
		Model:    c.config.Model,
		Messages: convertMessages(messages),
	}

	if c.config.ParallelToolCalls != nil {
		request.ParallelToolCalls = *c.config.ParallelToolCalls
	}

	response, err := c.client.CreateChatCompletion(ctx, request)
	if err != nil {
		c.logger.Error("OpenAI chat completion failed", "error", err, "model", c.config.Model)
		return "", err
	}

	c.logger.Debug("OpenAI chat completion successful", "model", c.config.Model, "response_length", len(response.Choices[0].Message.Content))
	return response.Choices[0].Message.Content, nil
}
