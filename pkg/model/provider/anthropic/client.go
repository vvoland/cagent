package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/docker/cagent/pkg/chat"
	latest "github.com/docker/cagent/pkg/config/v1"
	"github.com/docker/cagent/pkg/desktop"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/tools"
)

// Client represents an Anthropic client wrapper implementing provider.Provider
// It holds the anthropic client and model config
type Client struct {
	client anthropic.Client
	config *latest.ModelConfig
	logger *slog.Logger
}

// NewClient creates a new Anthropic client from the provided configuration
func NewClient(cfg *latest.ModelConfig, env environment.Provider, logger *slog.Logger, opts ...options.Opt) (*Client, error) {
	if cfg == nil {
		logger.Error("Anthropic client creation failed", "error", "model configuration is required")
		return nil, errors.New("model configuration is required")
	}

	if cfg.Provider != "anthropic" {
		logger.Error("Anthropic client creation failed", "error", "model type must be 'anthropic'", "actual_type", cfg.Provider)
		return nil, errors.New("model type must be 'anthropic'")
	}

	var globalOptions options.ModelOptions
	for _, opt := range opts {
		opt(&globalOptions)
	}

	var requestOptions []option.RequestOption
	if gateway := globalOptions.Gateway(); gateway == "" {
		authToken, err := env.Get(context.TODO(), "ANTHROPIC_API_KEY")
		if err != nil || authToken == "" {
			logger.Error("Anthropic client creation failed", "error", "failed to get authentication token", "details", err)
			return nil, errors.New("ANTHROPIC_API_KEY environment variable is required")
		}

		requestOptions = append(requestOptions,
			option.WithAPIKey(authToken),
		)
	} else {
		authToken := desktop.GetToken(context.TODO())
		if authToken == "" {
			logger.Error("Anthropic client creation failed", "error", "failed to get Docker Desktop's authentication token")
			return nil, errors.New("sorry, you first need to sign in Docker Desktop to use the Docker AI Gateway")
		}

		requestOptions = append(requestOptions,
			option.WithAuthToken(authToken),
			option.WithAPIKey(authToken),
			option.WithBaseURL(gateway),
		)
	}

	logger.Debug("Anthropic API key found, creating client")
	client := anthropic.NewClient(requestOptions...)
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

	maxTokens := int64(c.config.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 8192
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.config.Model),
		MaxTokens: maxTokens,
		Messages:  convertMessages(messages),
		Tools:     convertTools(requestTools),
	}

	if len(requestTools) > 0 {
		c.logger.Debug("Adding tools to Anthropic request", "tool_count", len(requestTools))
	}

	// Log the request details for debugging
	c.logger.Debug("Anthropic chat completion stream request",
		"model", params.Model,
		"max_tokens", maxTokens,
		"message_count", len(params.Messages))

	if c.logger.Enabled(ctx, slog.LevelDebug) {
		b, err := json.Marshal(params)
		if err != nil {
			c.logger.Error("Failed to marshal Anthropic request", "error", err)
		}
		c.logger.Debug("Request", "request", string(b))
	}

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
				blockLen := len(msg.ToolCalls)
				msgContent := strings.TrimSpace(msg.Content)
				offset := 0
				if msgContent != "" {
					blockLen++
					offset = 1
				}
				toolUseBlocks := make([]anthropic.ContentBlockParamUnion, blockLen)
				if msgContent != "" {
					toolUseBlocks[0] = anthropic.NewTextBlock(msgContent)
				}
				for j, toolCall := range msg.ToolCalls {
					var inpts map[string]any
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &inpts); err != nil {
						inpts = map[string]any{}
					}
					toolUseBlocks[j+offset] = anthropic.ContentBlockParamUnion{
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
			toolResult := anthropic.NewToolResultBlock(msg.ToolCallID, strings.TrimSpace(msg.Content), false)
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
