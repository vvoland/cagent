package openai

import (
	"context"
	"errors"
	"os"

	"github.com/rumpl/cagent/config"
	"github.com/sashabaranov/go-openai"
)

// Client represents an OpenAI client wrapper
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

// NewClientFromConfig creates a new OpenAI client from the configuration by model name
func NewClientFromConfig(cfg *config.Config, modelName string) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("configuration is required")
	}

	modelCfg, err := cfg.GetModelConfig(modelName)
	if err != nil {
		return nil, err
	}

	return NewClient(modelCfg)
}

// CreateChatCompletionStream creates a streaming chat completion request
// It returns a stream that can be iterated over to get completion chunks
func (c *Client) CreateChatCompletionStream(
	ctx context.Context,
	messages []openai.ChatCompletionMessage,
	tools []openai.Tool,
) (*openai.ChatCompletionStream, error) {
	if len(messages) == 0 {
		return nil, errors.New("at least one message is required")
	}

	// Create request with config parameters
	request := openai.ChatCompletionRequest{
		Model:            c.config.Model,
		Messages:         messages,
		Temperature:      float32(c.config.Temperature),
		MaxTokens:        c.config.MaxTokens,
		TopP:             float32(c.config.TopP),
		FrequencyPenalty: float32(c.config.FrequencyPenalty),
		PresencePenalty:  float32(c.config.PresencePenalty),
		Stream:           true,
	}

	// Add tools if provided
	if len(tools) > 0 {
		request.Tools = tools
	}

	// Log the request in JSON format for debugging
	// if requestJSON, err := json.MarshalIndent(request, "", "  "); err == nil {
	// 	fmt.Printf("Chat completion request:\n%s\n", string(requestJSON))
	// } else {
	// 	fmt.Printf("Error marshaling request to JSON: %v\n", err)
	// }

	// Create the stream
	return c.client.CreateChatCompletionStream(ctx, request)
}
