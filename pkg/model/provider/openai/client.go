package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/tools"
	"github.com/sashabaranov/go-openai"
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
}

// NewClient creates a new OpenAI client from the provided configuration
func NewClient(cfg *config.ModelConfig) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("model configuration is required")
	}

	if cfg.Type != "openai" {
		return nil, errors.New("model type must be 'openai'")
	}

	// Get the API key from environment variables
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable is required")
	}

	// Create a client config
	clientConfig := openai.DefaultConfig(apiKey)

	// Create the OpenAI client
	client := openai.NewClientWithConfig(clientConfig)

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// GetClient returns the underlying OpenAI client
func (c *Client) GetClient() *openai.Client {
	return c.client
}

// GetConfig returns the model configuration
func (c *Client) GetConfig() *config.ModelConfig {
	return c.config
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
			Role: msg.Role,
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
	tools []tools.Tool,
) (chat.MessageStream, error) {
	if len(messages) == 0 {
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

	if c.config.MaxTokens > 0 {
		request.MaxTokens = c.config.MaxTokens
	}

	if len(tools) > 0 {
		request.Tools = make([]openai.Tool, len(tools))
		for i, tool := range tools {
			request.Tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Strict:      tool.Function.Strict,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
	}

	// Log the request in JSON format for debugging
	if requestJSON, err := json.MarshalIndent(request, "", "  "); err == nil {
		fmt.Printf("Chat completion request:\n%s\n", string(requestJSON))
	} else {
		fmt.Printf("Error marshaling request to JSON: %v\n", err)
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return nil, err
	}

	return &StreamAdapter{stream: stream}, nil
}
