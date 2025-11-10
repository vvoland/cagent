package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"

	"github.com/docker/cagent/pkg/chat"
	latest "github.com/docker/cagent/pkg/config/v2"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/httpclient"
	"github.com/docker/cagent/pkg/model/provider/base"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/tools"
)

// Client represents an OpenAI client wrapper
// It implements the provider.Provider interface
type Client struct {
	base.Config
	clientFn func(context.Context) (*openai.Client, error)
}

// NewClient creates a new OpenAI client from the provided configuration
func NewClient(ctx context.Context, cfg *latest.ModelConfig, env environment.Provider, opts ...options.Opt) (*Client, error) {
	if cfg == nil {
		slog.Error("OpenAI client creation failed", "error", "model configuration is required")
		return nil, errors.New("model configuration is required")
	}

	var globalOptions options.ModelOptions
	for _, opt := range opts {
		opt(&globalOptions)
	}

	var clientFn func(context.Context) (*openai.Client, error)
	if gateway := globalOptions.Gateway(); gateway == "" {
		key := cfg.TokenKey
		if key == "" {
			key = "OPENAI_API_KEY"
		}
		authToken := env.Get(ctx, key)
		if authToken == "" {
			return nil, fmt.Errorf("%s environment variable is required", key)
		}

		var clientOptions []option.RequestOption
		clientOptions = append(clientOptions, option.WithAPIKey(authToken))

		if cfg.Provider == "azure" {
			// Azure configuration
			if cfg.BaseURL != "" {
				clientOptions = append(clientOptions, option.WithBaseURL(cfg.BaseURL))
			}

			// Azure API version from provider opts
			if cfg.ProviderOpts != nil {
				if apiVersion, exists := cfg.ProviderOpts["api_version"]; exists {
					slog.Debug("Setting API version", "api_version", apiVersion)
					if apiVersionStr, ok := apiVersion.(string); ok {
						clientOptions = append(clientOptions, option.WithHeader("api-version", apiVersionStr))
					}
				}
			}
		} else if cfg.BaseURL != "" {
			clientOptions = append(clientOptions, option.WithBaseURL(cfg.BaseURL))
		}

		httpClient := httpclient.NewHTTPClient()
		clientOptions = append(clientOptions, option.WithHTTPClient(httpClient))

		slog.Debug("OpenAI API key found, creating client")
		client := openai.NewClient(clientOptions...)
		clientFn = func(context.Context) (*openai.Client, error) {
			return &client, nil
		}
	} else {
		// Fail fast if Docker Desktop's auth token isn't available
		if env.Get(ctx, environment.DockerDesktopTokenEnv) == "" {
			slog.Error("OpenAI client creation failed", "error", "failed to get Docker Desktop's authentication token")
			return nil, errors.New("sorry, you first need to sign in Docker Desktop to use the Docker AI Gateway")
		}

		// When using a Gateway, tokens are short-lived.
		clientFn = func(ctx context.Context) (*openai.Client, error) {
			// Query a fresh auth token each time the client is used
			authToken := env.Get(ctx, environment.DockerDesktopTokenEnv)
			if authToken == "" {
				return nil, errors.New("failed to get Docker Desktop token for Gateway")
			}

			var clientOptions []option.RequestOption
			clientOptions = append(clientOptions, option.WithAPIKey(authToken), option.WithBaseURL(gateway+"/v1"))

			httpClient := httpclient.NewHTTPClient(
				httpclient.WithProxiedBaseURL(defaultsTo(cfg.BaseURL, "https://api.openai.com/v1")),
				httpclient.WithProvider(cfg.Provider),
				httpclient.WithModel(cfg.Model),
			)
			clientOptions = append(clientOptions, option.WithHTTPClient(httpClient))

			client := openai.NewClient(clientOptions...)
			return &client, nil
		}
	}

	slog.Debug("OpenAI client created successfully", "model", cfg.Model)

	return &Client{
		Config: base.Config{
			ModelConfig:  *cfg,
			ModelOptions: globalOptions,
			Env:          env,
		},
		clientFn: clientFn,
	}, nil
}

func convertMultiContent(multiContent []chat.MessagePart) []openai.ChatCompletionContentPartUnionParam {
	parts := make([]openai.ChatCompletionContentPartUnionParam, len(multiContent))
	for i, part := range multiContent {
		switch part.Type {
		case chat.MessagePartTypeText:
			parts[i] = openai.TextContentPart(part.Text)
		case chat.MessagePartTypeImageURL:
			if part.ImageURL != nil {
				parts[i] = openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL:    part.ImageURL.URL,
					Detail: string(part.ImageURL.Detail),
				})
			}
		}
	}
	return parts
}

// convertMessages converts chat.ChatCompletionMessage to openai.ChatCompletionMessageParamUnion
func convertMessages(messages []chat.Message) []openai.ChatCompletionMessageParamUnion {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for i := range messages {
		msg := &messages[i]

		// Skip invalid assistant messages upfront. This can happen if the model is out of tokens (max_tokens reached)
		if msg.Role == chat.MessageRoleAssistant && len(msg.ToolCalls) == 0 && len(msg.MultiContent) == 0 && strings.TrimSpace(msg.Content) == "" {
			continue
		}

		var openaiMessage openai.ChatCompletionMessageParamUnion

		switch msg.Role {
		case chat.MessageRoleSystem:
			if len(msg.MultiContent) == 0 {
				openaiMessage = openai.SystemMessage(msg.Content)
			} else {
				// Convert multi-content for system messages
				textParts := make([]openai.ChatCompletionContentPartTextParam, 0)
				for _, part := range msg.MultiContent {
					if part.Type == chat.MessagePartTypeText {
						textParts = append(textParts, openai.ChatCompletionContentPartTextParam{
							Text: part.Text,
						})
					}
				}
				openaiMessage = openai.SystemMessage(textParts)
			}

		case chat.MessageRoleUser:
			if len(msg.MultiContent) == 0 {
				openaiMessage = openai.UserMessage(msg.Content)
			} else {
				openaiMessage = openai.UserMessage(convertMultiContent(msg.MultiContent))
			}

		case chat.MessageRoleAssistant:
			assistantParam := openai.ChatCompletionAssistantMessageParam{}

			if len(msg.MultiContent) == 0 {
				if msg.Content != "" {
					assistantParam.Content.OfString = param.NewOpt(msg.Content)
				}
			} else {
				// Convert multi-content for assistant messages
				contentParts := make([]openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion, 0)
				for _, part := range msg.MultiContent {
					if part.Type == chat.MessagePartTypeText {
						contentParts = append(contentParts, openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
							OfText: &openai.ChatCompletionContentPartTextParam{
								Text: part.Text,
							},
						})
					}
				}
				if len(contentParts) > 0 {
					assistantParam.Content.OfArrayOfContentParts = contentParts
				}
			}

			if msg.Name != "" {
				assistantParam.Name = param.NewOpt(msg.Name)
			}

			if msg.FunctionCall != nil {
				assistantParam.FunctionCall.Name = msg.FunctionCall.Name           //nolint:staticcheck // deprecated but still needed for compatibility
				assistantParam.FunctionCall.Arguments = msg.FunctionCall.Arguments //nolint:staticcheck // deprecated but still needed for compatibility
			}

			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, len(msg.ToolCalls))
				for j, toolCall := range msg.ToolCalls {
					toolCalls[j] = openai.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
							ID: toolCall.ID,
							Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
								Name:      toolCall.Function.Name,
								Arguments: toolCall.Function.Arguments,
							},
						},
					}
				}
				assistantParam.ToolCalls = toolCalls
			}

			openaiMessage.OfAssistant = &assistantParam

		case chat.MessageRoleTool:
			toolParam := openai.ChatCompletionToolMessageParam{
				ToolCallID: msg.ToolCallID,
			}

			if len(msg.MultiContent) == 0 {
				toolParam.Content.OfString = param.NewOpt(msg.Content)
			} else {
				// Convert multi-content for tool messages
				textParts := make([]openai.ChatCompletionContentPartTextParam, 0)
				for _, part := range msg.MultiContent {
					if part.Type == chat.MessagePartTypeText {
						textParts = append(textParts, openai.ChatCompletionContentPartTextParam{
							Text: part.Text,
						})
					}
				}
				toolParam.Content.OfArrayOfContentParts = textParts
			}

			openaiMessage.OfTool = &toolParam
		}

		openaiMessages = append(openaiMessages, openaiMessage)
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
	slog.Debug("Creating OpenAI chat completion stream",
		"model", c.ModelConfig.Model,
		"message_count", len(messages),
		"tool_count", len(requestTools))

	if len(messages) == 0 {
		slog.Error("OpenAI stream creation failed", "error", "at least one message is required")
		return nil, errors.New("at least one message is required")
	}

	trackUsage := c.ModelConfig.TrackUsage == nil || *c.ModelConfig.TrackUsage

	params := openai.ChatCompletionNewParams{
		Model:    c.ModelConfig.Model,
		Messages: convertMessages(messages),
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(trackUsage),
		},
	}

	if c.ModelConfig.Temperature != nil {
		params.Temperature = openai.Float(*c.ModelConfig.Temperature)
	}
	if c.ModelConfig.TopP != nil {
		params.TopP = openai.Float(*c.ModelConfig.TopP)
	}
	if c.ModelConfig.FrequencyPenalty != nil {
		params.FrequencyPenalty = openai.Float(*c.ModelConfig.FrequencyPenalty)
	}
	if c.ModelConfig.PresencePenalty != nil {
		params.PresencePenalty = openai.Float(*c.ModelConfig.PresencePenalty)
	}

	if maxToken := c.ModelConfig.MaxTokens; maxToken > 0 {
		if !isResponsesOnlyModel(c.ModelConfig.Model) {
			params.MaxTokens = openai.Int(int64(maxToken))
			slog.Debug("OpenAI request configured with max tokens", "max_tokens", maxToken, "model", c.ModelConfig.Model)
		} else {
			params.MaxCompletionTokens = openai.Int(int64(maxToken))
			slog.Debug("using max_completion_tokens instead of max_tokens for Responses-API models", "model", c.ModelConfig.Model)
		}
	}

	if len(requestTools) > 0 {
		slog.Debug("Adding tools to OpenAI request", "tool_count", len(requestTools))
		toolsParam := make([]openai.ChatCompletionToolUnionParam, len(requestTools))
		for i, tool := range requestTools {
			parameters, err := ConvertParametersToSchema(tool.Parameters)
			if err != nil {
				slog.Debug("Failed to convert tool parameters to OpenAI schema", "tool_name", tool.Name, "error", err)
				return nil, err
			}

			paramsMap, ok := parameters.(map[string]any)
			if !ok {
				slog.Error("Converted parameters is not a map", "tool", tool.Name)
				return nil, fmt.Errorf("converted parameters is not a map for tool %s", tool.Name)
			}

			toolsParam[i] = openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  shared.FunctionParameters(paramsMap),
			})

			slog.Debug("Added tool to OpenAI request", "tool_name", tool.Name)
		}
		params.Tools = toolsParam

		if c.ModelConfig.ParallelToolCalls != nil {
			params.ParallelToolCalls = openai.Bool(*c.ModelConfig.ParallelToolCalls)
		}
	}

	// Apply thinking budget: set reasoning_effort parameter
	if c.ModelConfig.ThinkingBudget != nil {
		effort, err := getOpenAIReasoningEffort(&c.ModelConfig)
		if err != nil {
			slog.Error("OpenAI request using thinking_budget failed", "error", err)
			return nil, err
		}
		params.ReasoningEffort = shared.ReasoningEffort(effort)
		slog.Debug("OpenAI request using thinking_budget", "reasoning_effort", effort)
	}

	// Apply structured output configuration
	if structuredOutput := c.ModelOptions.StructuredOutput(); structuredOutput != nil {
		slog.Debug("OpenAI request using structured output", "name", structuredOutput.Name, "strict", structuredOutput.Strict)

		params.ResponseFormat.OfJSONSchema = &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        structuredOutput.Name,
				Description: openai.String(structuredOutput.Description),
				Schema:      jsonSchema(structuredOutput.Schema),
				Strict:      openai.Bool(structuredOutput.Strict),
			},
		}
	}

	// Log the request in JSON format for debugging
	if requestJSON, err := json.Marshal(params); err == nil {
		slog.Debug("OpenAI chat completion request", "request", string(requestJSON))
	} else {
		slog.Error("Failed to marshal OpenAI request to JSON", "error", err)
	}

	client, err := c.clientFn(ctx)
	if err != nil {
		slog.Error("Failed to create OpenAI client", "error", err)
		return nil, err
	}

	stream := client.Chat.Completions.NewStreaming(ctx, params)

	slog.Debug("OpenAI chat completion stream created successfully", "model", c.ModelConfig.Model)
	return newStreamAdapter(stream, trackUsage), nil
}

// ConvertParametersToSchema converts parameters to OpenAI Schema format
func ConvertParametersToSchema(params any) (any, error) {
	return tools.SchemaToMap(params)
}

// isResponsesOnlyModel returns true for newer OpenAI models that use the Responses API
// and expect max_completion_tokens/max_output_tokens instead of max_tokens
func isResponsesOnlyModel(model string) bool {
	m := strings.ToLower(model)
	if strings.HasPrefix(m, "gpt-4.1") {
		return true
	}
	if strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") {
		return true
	}
	if strings.HasPrefix(m, "gpt-5") {
		return true
	}
	return false
}

func isOpenAIReasoningModel(model string) bool {
	m := strings.ToLower(model)
	if strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") {
		return true
	}
	if strings.HasPrefix(m, "gpt-5") {
		return true
	}
	return false
}

// getOpenAIReasoningEffort resolves the reasoning effort value from the
// model configuration's ThinkingBudget. Returns the effort (minimal|low|medium|high) or an error
func getOpenAIReasoningEffort(cfg *latest.ModelConfig) (effort string, err error) {
	if cfg == nil || cfg.ThinkingBudget == nil {
		return "", nil
	}

	if !isOpenAIReasoningModel(cfg.Model) {
		slog.Warn("OpenAI reasoning effort is not supported for this model, ignoring thinking_budget", "model", cfg.Model)
		return "", nil
	}

	effort = strings.TrimSpace(strings.ToLower(cfg.ThinkingBudget.Effort))
	if effort == "minimal" || effort == "low" || effort == "medium" || effort == "high" {
		return effort, nil
	}

	return "", fmt.Errorf("OpenAI requests only support 'minimal', 'low', 'medium', 'high' as values for thinking_budget effort, got effort: '%s', tokens: '%d'", effort, cfg.ThinkingBudget.Tokens)
}

// jsonSchema is a helper type that implements json.Marshaler for map[string]any
// This allows us to pass schema maps to the OpenAI library which expects json.Marshaler
type jsonSchema map[string]any

func (j jsonSchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any(j))
}

func defaultsTo(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}
