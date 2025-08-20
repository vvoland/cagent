package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/docker/cagent/pkg/chat"
	latest "github.com/docker/cagent/pkg/config/v1"
	"github.com/docker/cagent/pkg/desktop"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/tools"
	"google.golang.org/genai"
)

// Client represents a Gemini client wrapper
// It implements the provider.Provider interface
type Client struct {
	client *genai.Client
	config *latest.ModelConfig
	logger *slog.Logger
}

// NewClient creates a new Gemini client from the provided configuration
func NewClient(ctx context.Context, cfg *latest.ModelConfig, env environment.Provider, logger *slog.Logger, opts ...options.Opt) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("model configuration is required")
	}

	if cfg.Provider != "gemini" {
		return nil, errors.New("model type must be 'gemini'")
	}

	var modelOptions options.ModelOptions
	for _, opt := range opts {
		opt(&modelOptions)
	}

	var apiKey string
	var httpOptions genai.HTTPOptions
	if gateway := modelOptions.Gateway(); gateway == "" {
		var err error
		apiKey, err = env.Get(ctx, "GOOGLE_API_KEY")
		if err != nil || apiKey == "" {
			return nil, errors.New("GOOGLE_API_KEY environment variable is required")
		}
	} else {
		// genai client requires a non-empty API key
		apiKey = desktop.GetToken(ctx)
		if apiKey == "" {
			return nil, errors.New("sorry, you first need to sign in Docker Desktop to use the Docker AI Gateway")
		}
		httpOptions.BaseURL = gateway
		httpOptions.Headers = make(http.Header)
		httpOptions.Headers.Set("Authorization", "Bearer "+apiKey)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:      apiKey,
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: httpOptions,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// convertMessagesToGemini converts chat.Messages into Gemini Contents
func convertMessagesToGemini(messages []chat.Message) []*genai.Content {
	contents := make([]*genai.Content, 0, len(messages))
	for i := range messages {
		msg := &messages[i]

		// Skip empty messages
		if msg.Content == "" && len(msg.MultiContent) == 0 && len(msg.ToolCalls) == 0 && msg.ToolCallID == "" {
			continue
		}

		var role genai.Role
		switch msg.Role {
		case chat.MessageRoleAssistant:
			role = genai.RoleModel
		default:
			role = genai.RoleUser
		}

		// Handle tool responses
		if msg.Role == chat.MessageRoleTool && msg.ToolCallID != "" {
			// Create a function response part
			part := genai.NewPartFromFunctionResponse(msg.ToolCallID, map[string]any{
				"result": msg.Content,
			})
			contents = append(contents, genai.NewContentFromParts([]*genai.Part{part}, role))
			continue
		}

		// Handle assistant messages with tool calls
		if msg.Role == chat.MessageRoleAssistant && len(msg.ToolCalls) > 0 {
			parts := make([]*genai.Part, 0)

			// Add text content if present
			if msg.Content != "" {
				parts = append(parts, genai.NewPartFromText(msg.Content))
			}

			// Add function calls
			for _, tc := range msg.ToolCalls {
				// Parse arguments from JSON string to map
				var args map[string]any
				if tc.Function.Arguments != "" {
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				}

				parts = append(parts, genai.NewPartFromFunctionCall(tc.Function.Name, args))
			}

			contents = append(contents, genai.NewContentFromParts(parts, role))
			continue
		}

		// Handle regular messages
		if len(msg.MultiContent) > 0 {
			parts := make([]*genai.Part, 0, len(msg.MultiContent))
			for _, part := range msg.MultiContent {
				if part.Type == chat.MessagePartTypeText {
					parts = append(parts, genai.NewPartFromText(part.Text))
				} else if part.Type == chat.MessagePartTypeImageURL && part.ImageURL != nil {
					// TODO: Implement image support for Gemini SDK
					// For now, add a text note about the image
					parts = append(parts, genai.NewPartFromText("[Image content not yet supported in Gemini provider]"))
				}
			}
			if len(parts) > 0 {
				contents = append(contents, genai.NewContentFromParts(parts, role))
			}
		} else if msg.Content != "" {
			contents = append(contents, genai.NewContentFromText(msg.Content, role))
		}
	}
	return contents
}

// buildConfig creates GenerateContentConfig from model config
func (c *Client) buildConfig() *genai.GenerateContentConfig {
	if c.config == nil {
		return nil
	}

	config := &genai.GenerateContentConfig{
		Temperature:      genai.Ptr(float32(c.config.Temperature)),
		TopP:             genai.Ptr(float32(c.config.TopP)),
		FrequencyPenalty: genai.Ptr(float32(c.config.FrequencyPenalty)),
		PresencePenalty:  genai.Ptr(float32(c.config.PresencePenalty)),
	}
	if c.config.MaxTokens > 0 {
		config.MaxOutputTokens = int32(c.config.MaxTokens)
	}
	return config
}

// convertToolsToGemini converts tools to Gemini format
func convertToolsToGemini(requestTools []tools.Tool) []*genai.Tool {
	if len(requestTools) == 0 {
		return nil
	}

	funcs := make([]*genai.FunctionDeclaration, 0, len(requestTools))
	for _, tool := range requestTools {
		funcs = append(funcs, &genai.FunctionDeclaration{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  convertParametersToSchema(tool.Function.Parameters),
		})
	}

	return []*genai.Tool{{
		FunctionDeclarations: funcs,
	}}
}

// convertParametersToSchema converts parameters to Gemini Schema format
func convertParametersToSchema(params any) *genai.Schema {
	if params == nil {
		return nil
	}

	// Convert FunctionParameters to Schema
	if funcParams, ok := params.(tools.FunctionParamaters); ok {
		// Convert type string to Gemini Type
		var schemaType genai.Type
		switch funcParams.Type {
		case "object":
			schemaType = genai.TypeObject
		case "string":
			schemaType = genai.TypeString
		case "number":
			schemaType = genai.TypeNumber
		case "integer":
			schemaType = genai.TypeInteger
		case "boolean":
			schemaType = genai.TypeBoolean
		case "array":
			schemaType = genai.TypeArray
		default:
			schemaType = genai.TypeObject
		}

		schema := &genai.Schema{
			Type:     schemaType,
			Required: funcParams.Required,
		}

		// Convert properties map
		if len(funcParams.Properties) > 0 {
			schema.Properties = make(map[string]*genai.Schema)
			for name := range funcParams.Properties {
				// Parse each property schema
				if propMap, ok := funcParams.Properties[name].(map[string]any); ok {
					propSchema := &genai.Schema{}
					if propType, ok := propMap["type"].(string); ok {
						switch propType {
						case "string":
							propSchema.Type = genai.TypeString
						case "number":
							propSchema.Type = genai.TypeNumber
						case "integer":
							propSchema.Type = genai.TypeInteger
						case "boolean":
							propSchema.Type = genai.TypeBoolean
						case "array":
							propSchema.Type = genai.TypeArray
							propSchema.Items = &genai.Schema{
								Type: genai.TypeString,
							}
						case "object":
							propSchema.Type = genai.TypeObject
						default:
							propSchema.Type = genai.TypeString
						}
					}
					if propDesc, ok := propMap["description"].(string); ok {
						propSchema.Description = propDesc
					}
					schema.Properties[name] = propSchema
				} else {
					// Default to string type
					schema.Properties[name] = &genai.Schema{
						Type: genai.TypeString,
					}
				}
			}
		}

		return schema
	}

	// Fallback for other parameter types
	return &genai.Schema{
		Type: genai.TypeObject,
	}
}

// CreateChatCompletionStream creates a streaming chat completion request
func (c *Client) CreateChatCompletionStream(
	ctx context.Context,
	messages []chat.Message,
	requestTools []tools.Tool,
) (chat.MessageStream, error) {
	if len(messages) == 0 {
		return nil, errors.New("at least one message is required")
	}

	config := c.buildConfig()

	// Add tools to config if provided
	if len(requestTools) > 0 {
		config.Tools = convertToolsToGemini(requestTools)

		// Enable function calling
		config.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAuto,
			},
		}

		// Debug: Log the tools we're sending
		c.logger.Debug("Gemini tools config", "tools", config.Tools)
		for _, tool := range config.Tools {
			for _, fn := range tool.FunctionDeclarations {
				c.logger.Debug("Function", "name", fn.Name, "desc", fn.Description, "params", fn.Parameters)
			}
		}
	}

	contents := convertMessagesToGemini(messages)

	// Debug: Log the messages we're sending
	c.logger.Debug("Gemini messages", "count", len(contents))
	for i, content := range contents {
		c.logger.Debug("Message", "index", i, "role", content.Role)
	}

	iter := c.client.Models.GenerateContentStream(ctx, c.config.Model, contents, config)
	return NewStreamAdapter(iter, c.config.Model, c.logger), nil
}

// CreateChatCompletion creates a non-streaming chat completion
func (c *Client) CreateChatCompletion(
	ctx context.Context,
	messages []chat.Message,
) (string, error) {
	result, err := c.client.Models.GenerateContent(ctx, c.config.Model, convertMessagesToGemini(messages), c.buildConfig())
	if err != nil {
		return "", err
	}

	// Check if there are function calls in the response
	if funcs := result.FunctionCalls(); len(funcs) > 0 {
		// For now, we'll return an error indicating function calls are not supported in non-streaming mode
		// This matches the behavior of other providers that expect streaming for tool use
		return "", errors.New("function calls are not supported in non-streaming mode, use streaming mode instead")
	}

	// Extract text content safely
	var textParts []string
	for _, candidate := range result.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					textParts = append(textParts, part.Text)
				}
			}
		}
	}

	if len(textParts) == 0 {
		return "", nil
	}

	return textParts[0], nil
}
