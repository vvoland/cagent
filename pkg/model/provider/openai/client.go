package openai

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/sashabaranov/go-openai"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/tools"
)

// StreamAdapter adapts the OpenAI stream to our interface
type StreamAdapter struct {
	stream *openai.ChatCompletionStream
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

	// Convert the choices
	for i := range openaiResponse.Choices {
		choice := &openaiResponse.Choices[i]
		response.Choices[i] = chat.MessageStreamChoice{
			Index:        choice.Index,
			FinishReason: chat.FinishReason(choice.FinishReason),
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

// Client represents an OpenAI client wrapper
// It implements the provider.Provider interface
type Client struct {
	client *openai.Client
	config *config.ModelConfig
	logger *slog.Logger
}

// NewClient creates a new OpenAI client from the provided configuration
func NewClient(cfg *config.ModelConfig, env environment.Provider, logger *slog.Logger) (*Client, error) {
	if cfg == nil {
		logger.Error("OpenAI client creation failed", "error", "model configuration is required")
		return nil, errors.New("model configuration is required")
	}

	if cfg.Type != "openai" {
		logger.Error("OpenAI client creation failed", "error", "model type must be 'openai'", "actual_type", cfg.Type)
		return nil, errors.New("model type must be 'openai'")
	}

	// Get the API key from environment variables
	apiKey, err := env.Get(context.TODO(), "OPENAI_API_KEY")
	if err != nil {
		logger.Error("OpenAI client creation failed", "error", "failed to get OPENAI_API_KEY from environment", "details", err)
		return nil, errors.New("OPENAI_API_KEY environment variable is required")
	}
	if apiKey == "" {
		logger.Error("OpenAI client creation failed", "error", "OPENAI_API_KEY environment variable is required")
		return nil, errors.New("OPENAI_API_KEY environment variable is required")
	}

	logger.Debug("OpenAI API key found, creating client")
	clientConfig := openai.DefaultConfig(apiKey)
	client := openai.NewClientWithConfig(clientConfig)
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
	if requestJSON, err := json.MarshalIndent(request, "", "  "); err == nil {
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
		c.logger.Error("OpenAI chat completion failed", "error", err, "model", c.config.Model)
		return "", err
	}

	c.logger.Debug("OpenAI chat completion successful", "model", c.config.Model, "response_length", len(response.Choices[0].Message.Content))
	return response.Choices[0].Message.Content, nil
}
