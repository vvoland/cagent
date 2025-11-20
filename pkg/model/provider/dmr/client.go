package dmr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
	"golang.org/x/term"

	"github.com/docker/cagent/pkg/chat"
	latest "github.com/docker/cagent/pkg/config/v2"
	"github.com/docker/cagent/pkg/input"
	"github.com/docker/cagent/pkg/model/provider/base"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/tools"
)

const (
	// dmrInferencePrefix mirrors github.com/docker/model-runner/pkg/inference.InferencePrefix.
	dmrInferencePrefix = "/engines"
	// dmrExperimentalEndpointsPrefix mirrors github.com/docker/model-runner/pkg/inference.ExperimentalEndpointsPrefix.
	dmrExperimentalEndpointsPrefix = "/exp/vDD4.40"
)

// Client represents an DMR client wrapper
// It implements the provider.Provider interface
type Client struct {
	base.Config
	client  openai.Client
	baseURL string
}

// NewClient creates a new DMR client from the provided configuration
func NewClient(ctx context.Context, cfg *latest.ModelConfig, opts ...options.Opt) (*Client, error) {
	if cfg == nil {
		slog.Error("DMR client creation failed", "error", "model configuration is required")
		return nil, errors.New("model configuration is required")
	}

	if cfg.Provider != "dmr" {
		slog.Error("DMR client creation failed", "error", "model type must be 'dmr'", "actual_type", cfg.Provider)
		return nil, errors.New("model type must be 'dmr'")
	}

	var globalOptions options.ModelOptions
	for _, opt := range opts {
		opt(&globalOptions)
	}

	endpoint, engine, err := getDockerModelEndpointAndEngine(ctx)
	if err != nil {
		slog.Debug("docker model status query failed", "error", err)
	} else {
		// Auto-pull the model if needed
		if err := pullDockerModelIfNeeded(ctx, cfg.Model); err != nil {
			slog.Debug("docker model pull failed", "error", err)
			return nil, err
		}
	}

	baseURL, clientOptions := resolveDMRBaseURL(cfg, endpoint)

	clientOptions = append(clientOptions, option.WithBaseURL(baseURL), option.WithAPIKey("")) // DMR doesn't need auth

	// Build runtime flags from ModelConfig and engine
	contextSize, providerRuntimeFlags, specOpts := parseDMRProviderOpts(cfg)
	configFlags := buildRuntimeFlagsFromModelConfig(engine, cfg)
	finalFlags, warnings := mergeRuntimeFlagsPreferUser(configFlags, providerRuntimeFlags)
	for _, w := range warnings {
		slog.Warn(w)
	}
	slog.Debug("DMR provider_opts parsed", "model", cfg.Model, "context_size", contextSize, "runtime_flags", finalFlags, "speculative_opts", specOpts, "engine", engine)
	if err := configureDockerModel(ctx, cfg.Model, contextSize, finalFlags, specOpts); err != nil {
		slog.Debug("docker model configure skipped or failed", "error", err)
	}

	slog.Debug("DMR client created successfully", "model", cfg.Model, "base_url", baseURL)

	return &Client{
		Config: base.Config{
			ModelConfig:  *cfg,
			ModelOptions: globalOptions,
		},
		client:  openai.NewClient(clientOptions...),
		baseURL: baseURL,
	}, nil
}

func inContainer() bool {
	finfo, err := os.Stat("/.dockerenv")
	return err == nil && finfo.Mode().IsRegular()
}

// resolveDMRBaseURL determines the correct base URL and HTTP options to talk to
// Docker Model Runner, mirroring the behavior of the `docker model` CLI as
// closely as possible.
//
// High‑level rules:
//   - If the user explicitly configured a BaseURL or MODEL_RUNNER_HOST, use that.
//   - For Desktop endpoints (model-runner.docker.internal) on the host, route
//     through the Docker Engine experimental endpoints prefix over the Unix socket.
//   - For standalone / offload endpoints like http://172.17.0.1:12435/engines/v1/,
//     use localhost:<port>/engines/v1/ on the host, and the gateway IP:port inside containers.
//   - Keep a small compatibility workaround for the legacy http://:0/engines/v1/ endpoint.
func resolveDMRBaseURL(cfg *latest.ModelConfig, endpoint string) (string, []option.RequestOption) {
	var clientOptions []option.RequestOption

	if cfg != nil && cfg.BaseURL != "" {
		return cfg.BaseURL, clientOptions
	}
	if host := os.Getenv("MODEL_RUNNER_HOST"); host != "" {
		trimmed := strings.TrimRight(host, "/")
		return trimmed + dmrInferencePrefix + "/v1/", clientOptions
	}

	ep := strings.TrimSpace(endpoint)

	// Legacy bug workaround: old DMR versions <= 0.1.44 could report http://:0/engines/v1/.
	if ep == "http://:0/engines/v1/" {
		// Use the default port on localhost.
		return "http://127.0.0.1:12434" + dmrInferencePrefix + "/v1/", clientOptions
	}

	// If we don't have a usable endpoint, fall back to sensible defaults.
	if ep == "" {
		if inContainer() {
			// In a container with no endpoint info, assume Docker Desktop's internal host.
			return "http://model-runner.docker.internal" + dmrInferencePrefix + "/v1/", clientOptions
		}
		// On the host with no endpoint info: default to the standard local Moby port.
		return "http://127.0.0.1:12434" + dmrInferencePrefix + "/v1/", clientOptions
	}

	u, err := url.Parse(ep)
	if err != nil {
		slog.Debug("failed to parse DMR endpoint, falling back to defaults", "endpoint", ep, "error", err)
		if inContainer() {
			return "http://model-runner.docker.internal" + dmrInferencePrefix + "/v1/", clientOptions
		}
		return "http://127.0.0.1:12434" + dmrInferencePrefix + "/v1/", clientOptions
	}

	host := u.Hostname()
	port := u.Port()

	if host == "model-runner.docker.internal" && !inContainer() {
		// Build "http://_/exp/vDD4.40/engines/v1" using the shared experimental prefix.
		expPrefix := strings.TrimPrefix(dmrExperimentalEndpointsPrefix, "/")
		baseURL := fmt.Sprintf("http://_/%s%s/v1", expPrefix, dmrInferencePrefix)

		clientOptions = append(clientOptions, option.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", "/var/run/docker.sock")
				},
			},
		}))

		return baseURL, clientOptions
	}

	if port == "" {
		// Default port when the status output omits it.
		port = "12434"
	}

	if inContainer() {
		baseURL := ep
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		return baseURL, clientOptions
	}

	// Host case – always talk to localhost:<port>/engines/v1/, even if the status
	// endpoint uses a gateway IP like 172.17.0.1.
	baseURL := fmt.Sprintf("http://127.0.0.1:%s%s/v1/", port, dmrInferencePrefix)
	return baseURL, clientOptions
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

// mergeRuntimeFlagsPreferUser merges derived engine flags (the ones under `models:` e.g. `temperature`)
// and user-provided runtime flags (the ones under `models:provider_opts:runtime_flags:`).
// If both specify the same flag key (e.g., --temp), the `models:provider_opts:runtime_flags:` value is preferred and a warning is returned.
// The result preserves order: all non-conflicting derived flags first, followed by all user flags.
func mergeRuntimeFlagsPreferUser(derived, user []string) (out, warnings []string) {
	type flagKV struct {
		key  string
		val  *string
		orig []string
	}
	parse := func(tokens []string) []flagKV {
		var out []flagKV
		for i := 0; i < len(tokens); i++ {
			tok := tokens[i]
			if strings.HasPrefix(tok, "-") {
				// handle --key=value or -k=value
				if idx := strings.Index(tok, "="); idx != -1 {
					k := tok[:idx]
					v := tok[idx+1:]
					out = append(out, flagKV{key: k, val: &v, orig: []string{tok}})
					continue
				}
				// handle spaced value: --key value
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					v := tokens[i+1]
					out = append(out, flagKV{key: tok, val: &v, orig: []string{tok, v}})
					i++
				} else {
					// boolean/flag without value
					out = append(out, flagKV{key: tok, val: nil, orig: []string{tok}})
				}
			} else {
				// orphan value; keep as-is
				v := tok
				out = append(out, flagKV{key: tok, val: &v, orig: []string{tok}})
			}
		}
		return out
	}
	der := parse(derived)
	usr := parse(user)

	// Index derived by key
	derivedIdx := map[string]int{}
	for i, kv := range der {
		// Only index proper flags (start with '-') to avoid colliding with orphan values
		if strings.HasPrefix(kv.key, "-") {
			derivedIdx[kv.key] = i
		}
	}

	conflicts := map[string]bool{}
	for _, kv := range usr {
		if strings.HasPrefix(kv.key, "-") {
			if _, ok := derivedIdx[kv.key]; ok {
				conflicts[kv.key] = true
			}
		}
	}

	for i, kv := range der {
		if strings.HasPrefix(kv.key, "-") && conflicts[kv.key] {
			warnings = append(warnings, "Overriding runtime flag "+kv.key+" with value from provider_opts.runtime_flags")
			continue // skip derived conflicting flag
		}
		out = append(out, kv.orig...)
		// also append any non-flag or orphan tokens (they are already in kv.orig)
		_ = i
	}
	// Append all user flags at the end
	for _, kv := range usr {
		out = append(out, kv.orig...)
	}
	return out, warnings
}

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

	var mergedMessages []openai.ChatCompletionMessageParamUnion

	for i := 0; i < len(openaiMessages); i++ {
		currentMsg := openaiMessages[i]

		// Check if current message is system or user
		currentRole := currentMsg.GetRole()
		if currentRole != nil && (*currentRole == "system" || *currentRole == "user") {
			var mergedContent string
			var mergedMultiContent []openai.ChatCompletionContentPartUnionParam
			j := i

			for j < len(openaiMessages) {
				msgToMerge := openaiMessages[j]
				msgRole := msgToMerge.GetRole()
				if msgRole == nil || *msgRole != *currentRole {
					break
				}

				content := msgToMerge.GetContent()
				// Try to extract string content
				switch v := content.AsAny().(type) {
				case *string:
					if mergedContent != "" {
						mergedContent += "\n"
					}
					mergedContent += *v
				case *[]openai.ChatCompletionContentPartUnionParam:
					if v != nil {
						mergedMultiContent = append(mergedMultiContent, *v...)
					}
				case *[]openai.ChatCompletionContentPartTextParam:
					// Convert text parts to union params
					if v != nil {
						for _, textPart := range *v {
							mergedMultiContent = append(mergedMultiContent, openai.ChatCompletionContentPartUnionParam{
								OfText: &openai.ChatCompletionContentPartTextParam{
									Text: textPart.Text,
								},
							})
						}
					}
				}
				j++
			}

			var mergedMessage openai.ChatCompletionMessageParamUnion
			if *currentRole == "system" {
				if len(mergedMultiContent) == 0 {
					mergedMessage = openai.SystemMessage(mergedContent)
				} else {
					textParts := make([]openai.ChatCompletionContentPartTextParam, 0)
					for _, part := range mergedMultiContent {
						if part.OfText != nil {
							textParts = append(textParts, *part.OfText)
						}
					}
					mergedMessage = openai.SystemMessage(textParts)
				}
			} else {
				if len(mergedMultiContent) == 0 {
					mergedMessage = openai.UserMessage(mergedContent)
				} else {
					mergedMessage = openai.UserMessage(mergedMultiContent)
				}
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
func (c *Client) CreateChatCompletionStream(ctx context.Context, messages []chat.Message, requestTools []tools.Tool) (chat.MessageStream, error) {
	slog.Debug("Creating DMR chat completion stream",
		"model", c.ModelConfig.Model,
		"message_count", len(messages),
		"tool_count", len(requestTools),
		"base_url", c.baseURL,
	)

	if len(messages) == 0 {
		slog.Error("DMR stream creation failed", "error", "at least one message is required")
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

	// Only set ParallelToolCalls when tools are present; matches OpenAI provider behavior
	if len(requestTools) > 0 && c.ModelConfig.ParallelToolCalls != nil {
		params.ParallelToolCalls = openai.Bool(*c.ModelConfig.ParallelToolCalls)
	}

	if c.ModelConfig.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(c.ModelConfig.MaxTokens))
		slog.Debug("DMR request configured with max tokens", "max_tokens", c.ModelConfig.MaxTokens)
	}

	if len(requestTools) > 0 {
		slog.Debug("Adding tools to DMR request", "tool_count", len(requestTools))
		toolsParam := make([]openai.ChatCompletionToolUnionParam, len(requestTools))
		for i, tool := range requestTools {
			// DMR requires the `description` key to be present; ensure a non-empty value
			// NOTE(krissetto): workaround, remove when fixed upstream, this shouldn't be necceessary
			desc := tool.Description
			if desc == "" {
				desc = "Function " + tool.Name
			}

			parameters, err := ConvertParametersToSchema(tool.Parameters)
			if err != nil {
				slog.Error("Failed to convert tool parameters to DMR schema", "error", err, "tool", tool.Name)
				return nil, fmt.Errorf("failed to convert tool parameters to DMR schema for tool %s: %w", tool.Name, err)
			}

			paramsMap, ok := parameters.(map[string]any)
			if !ok {
				slog.Error("Converted parameters is not a map", "tool", tool.Name)
				return nil, fmt.Errorf("converted parameters is not a map for tool %s", tool.Name)
			}

			toolsParam[i] = openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(desc),
				Parameters:  shared.FunctionParameters(paramsMap),
			})
		}
		params.Tools = toolsParam

		if c.ModelConfig.ParallelToolCalls != nil {
			params.ParallelToolCalls = openai.Bool(*c.ModelConfig.ParallelToolCalls)
		}
	}

	// Log the request in JSON format for debugging
	if requestJSON, err := json.Marshal(params); err == nil {
		slog.Debug("DMR chat completion request", "request", string(requestJSON))
	} else {
		slog.Error("Failed to marshal DMR request to JSON", "error", err)
	}

	if structuredOutput := c.ModelOptions.StructuredOutput(); structuredOutput != nil {
		slog.Debug("Adding structured output to DMR request", "structured_output", structuredOutput)

		params.ResponseFormat.OfJSONSchema = &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        structuredOutput.Name,
				Description: openai.String(structuredOutput.Description),
				Schema:      jsonSchema(structuredOutput.Schema),
				Strict:      openai.Bool(structuredOutput.Strict),
			},
		}
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)

	slog.Debug("DMR chat completion stream created successfully", "model", c.ModelConfig.Model, "base_url", c.baseURL)
	return newStreamAdapter(stream, trackUsage), nil
}

// CreateEmbedding generates an embedding vector for the given text with usage tracking.
func (c *Client) CreateEmbedding(ctx context.Context, text string) (*base.EmbeddingResult, error) {
	slog.Debug("Creating DMR embedding", "model", c.ModelConfig.Model, "text_length", len(text), "base_url", c.baseURL)

	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model: c.ModelConfig.Model,
	}

	response, err := c.client.Embeddings.New(ctx, params)
	if err != nil {
		slog.Error("DMR embedding request failed", "error", err)
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned from DMR")
	}

	// Convert []float32 to []float64
	embedding32 := response.Data[0].Embedding
	embedding := make([]float64, len(embedding32))
	for i, v := range embedding32 {
		embedding[i] = float64(v)
	}

	// Extract usage information
	inputTokens := int(response.Usage.PromptTokens)
	totalTokens := int(response.Usage.TotalTokens)

	// DMR is local/free, so cost is 0
	cost := 0.0

	slog.Debug("DMR embedding created successfully",
		"dimension", len(embedding),
		"input_tokens", inputTokens,
		"total_tokens", totalTokens)

	return &base.EmbeddingResult{
		Embedding:   embedding,
		InputTokens: inputTokens,
		TotalTokens: totalTokens,
		Cost:        cost,
	}, nil
}

// CreateBatchEmbedding generates embedding vectors for multiple texts with usage tracking.
func (c *Client) CreateBatchEmbedding(ctx context.Context, texts []string) (*base.BatchEmbeddingResult, error) {
	if len(texts) == 0 {
		return &base.BatchEmbeddingResult{
			Embeddings:  [][]float64{},
			InputTokens: 0,
			TotalTokens: 0,
			Cost:        0,
		}, nil
	}

	slog.Debug("Creating DMR batch embeddings", "model", c.ModelConfig.Model, "batch_size", len(texts), "base_url", c.baseURL)

	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model: c.ModelConfig.Model,
	}

	response, err := c.client.Embeddings.New(ctx, params)
	if err != nil {
		slog.Error("DMR batch embedding request failed", "error", err)
		return nil, fmt.Errorf("failed to create batch embeddings: %w", err)
	}

	if len(response.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(response.Data))
	}

	// Convert embeddings from []float32 to [][]float64
	embeddings := make([][]float64, len(response.Data))
	for i, data := range response.Data {
		embedding32 := data.Embedding
		embedding := make([]float64, len(embedding32))
		for j, v := range embedding32 {
			embedding[j] = float64(v)
		}
		embeddings[i] = embedding
	}

	// Extract usage information
	inputTokens := int(response.Usage.PromptTokens)
	totalTokens := int(response.Usage.TotalTokens)

	// DMR is local/free, so cost is 0
	cost := 0.0

	slog.Debug("DMR batch embeddings created successfully",
		"batch_size", len(embeddings),
		"dimension", len(embeddings[0]),
		"input_tokens", inputTokens,
		"total_tokens", totalTokens)

	return &base.BatchEmbeddingResult{
		Embeddings:  embeddings,
		InputTokens: inputTokens,
		TotalTokens: totalTokens,
		Cost:        cost,
	}, nil
}

// ConvertParametersToSchema converts parameters to DMR Schema format
func ConvertParametersToSchema(params any) (any, error) {
	m, err := tools.SchemaToMap(params)
	if err != nil {
		return nil, err
	}

	// DMR models tend to dislike `additionalProperties` in the schema
	// e.g. ai/qwen3 and ai/gpt-oss
	delete(m, "additionalProperties")

	return m, nil
}

type speculativeDecodingOpts struct {
	draftModel     string
	numTokens      int
	acceptanceRate float64
}

func parseDMRProviderOpts(cfg *latest.ModelConfig) (contextSize int, runtimeFlags []string, specOpts *speculativeDecodingOpts) {
	if cfg == nil {
		return 0, nil, nil
	}

	// Context length is now sourced from the standard max_tokens field
	contextSize = cfg.MaxTokens

	slog.Debug("DMR provider opts", "provider_opts", cfg.ProviderOpts)

	if len(cfg.ProviderOpts) > 0 {
		if v, ok := cfg.ProviderOpts["runtime_flags"]; ok {
			switch t := v.(type) {
			case []any:
				for _, item := range t {
					runtimeFlags = append(runtimeFlags, fmt.Sprint(item))
				}
			case []string:
				runtimeFlags = append(runtimeFlags, t...)
			case string:
				parts := strings.Fields(strings.ReplaceAll(t, ",", " "))
				runtimeFlags = append(runtimeFlags, parts...)
			}
		}

		// Parse speculative decoding options
		var hasDraftModel, hasNumTokens, hasAcceptanceRate bool
		var draftModel string
		var numTokens int
		var acceptanceRate float64

		if v, ok := cfg.ProviderOpts["speculative_draft_model"]; ok {
			if s, ok := v.(string); ok && s != "" {
				draftModel = s
				hasDraftModel = true
			}
		}

		if v, ok := cfg.ProviderOpts["speculative_num_tokens"]; ok {
			switch t := v.(type) {
			case float64:
				numTokens = int(t)
				hasNumTokens = true
			case uint64:
				numTokens = int(t)
				hasNumTokens = true
			case string:
				s := strings.TrimSpace(t)
				if s != "" {
					if n, err := strconv.Atoi(s); err == nil {
						numTokens = n
						hasNumTokens = true
					} else if f, err := strconv.ParseFloat(s, 64); err == nil {
						numTokens = int(f)
						hasNumTokens = true
					}
				}
			}
		}

		if v, ok := cfg.ProviderOpts["speculative_acceptance_rate"]; ok {
			switch t := v.(type) {
			case float64:
				acceptanceRate = t
				hasAcceptanceRate = true
			case uint64:
				acceptanceRate = float64(t)
				hasAcceptanceRate = true
			case string:
				s := strings.TrimSpace(t)
				if s != "" {
					if f, err := strconv.ParseFloat(s, 64); err == nil {
						acceptanceRate = f
						hasAcceptanceRate = true
					}
				}
			}
		}

		// Only create specOpts if at least one field is set
		if hasDraftModel || hasNumTokens || hasAcceptanceRate {
			specOpts = &speculativeDecodingOpts{
				draftModel:     draftModel,
				numTokens:      numTokens,
				acceptanceRate: acceptanceRate,
			}
		}
	}

	return contextSize, runtimeFlags, specOpts
}

func pullDockerModelIfNeeded(ctx context.Context, model string) error {
	// Check if running in interactive mode (stdin is a terminal)
	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	if !interactive {
		// In non-interactive mode (CI / Servers), do not attempt to pull the model
		return nil
	}

	if modelExists(ctx, model) {
		slog.Debug("Model already exists, skipping pull", "model", model)
		return nil
	}

	// Prompt user for confirmation in interactive mode
	fmt.Printf("\nModel %s not found locally.\n", model)
	fmt.Printf("Do you want to pull it now? ([y]es/[n]o): ")

	response, err := input.ReadLine(ctx, os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("model pull declined by user")
	}

	// Pull the model
	slog.Info("Pulling DMR model", "model", model)
	fmt.Printf("Pulling model %s...\n", model)
	cmd := exec.CommandContext(ctx, "docker", "model", "pull", model)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull model %s: %w", model, err)
	}

	slog.Info("Model pulled successfully", "model", model)
	fmt.Printf("Model %s pulled successfully.\n", model)

	return nil
}

func modelExists(ctx context.Context, model string) bool {
	cmd := exec.CommandContext(ctx, "docker", "model", "inspect", model)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		slog.Debug("Model does not exist", "model", model, "error", strings.TrimSpace(stderr.String()))
		return false
	}
	return true
}

func configureDockerModel(ctx context.Context, model string, contextSize int, runtimeFlags []string, specOpts *speculativeDecodingOpts) error {
	args := buildDockerModelConfigureArgs(model, contextSize, runtimeFlags, specOpts)

	cmd := exec.CommandContext(ctx, "docker", args...)
	slog.Debug("Running docker model configure", "model", model, "args", args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(strings.TrimSpace(stderr.String()))
	}
	slog.Debug("docker model configure completed", "model", model)
	return nil
}

// buildDockerModelConfigureArgs returns the argument vector passed to `docker` for model configuration.
// It formats context size, speculative decoding options, and runtime flags consistently with the CLI contract.
func buildDockerModelConfigureArgs(model string, contextSize int, runtimeFlags []string, specOpts *speculativeDecodingOpts) []string {
	args := []string{"model", "configure"}
	if contextSize > 0 {
		args = append(args, "--context-size="+strconv.Itoa(contextSize))
	}
	if specOpts != nil {
		if specOpts.draftModel != "" {
			args = append(args, "--speculative-draft-model="+specOpts.draftModel)
		}
		if specOpts.numTokens > 0 {
			args = append(args, "--speculative-num-tokens="+strconv.Itoa(specOpts.numTokens))
		}
		if specOpts.acceptanceRate > 0 {
			args = append(args, "--speculative-min-acceptance-rate="+strconv.FormatFloat(specOpts.acceptanceRate, 'f', -1, 64))
		}
	}
	args = append(args, model)
	if len(runtimeFlags) > 0 {
		args = append(args, "--")
		args = append(args, runtimeFlags...)
	}
	return args
}

func getDockerModelEndpointAndEngine(ctx context.Context) (endpoint, engine string, err error) {
	cmd := exec.CommandContext(ctx, "docker", "model", "status", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", "", errors.New(strings.TrimSpace(stderr.String()))
	}

	type status struct {
		Running  bool              `json:"running"`
		Backends map[string]string `json:"backends"`
		Endpoint string            `json:"endpoint"`
		Engine   string            `json:"engine"`
	}
	var st status
	if err := json.Unmarshal(stdout.Bytes(), &st); err != nil {
		return "", "", err
	}

	endpoint = strings.TrimSpace(st.Endpoint)

	engine = strings.TrimSpace(st.Engine)
	if engine == "" {
		if st.Backends != nil {
			if _, ok := st.Backends["llama.cpp"]; ok {
				engine = "llama.cpp"
			} else {
				for k := range st.Backends {
					engine = k
					break
				}
			}
		}
	}
	if engine == "" {
		engine = "llama.cpp"
	}

	return endpoint, engine, nil
}

// buildRuntimeFlagsFromModelConfig converts standard ModelConfig fields into backend-specific
// runtime flags that the model-runner understands when launching the engine.
// Currently supports the default engine "llama.cpp". Unknown/unsupported fields are ignored.
func buildRuntimeFlagsFromModelConfig(engine string, cfg *latest.ModelConfig) []string {
	var flags []string
	if cfg == nil {
		return flags
	}
	eng := strings.TrimSpace(engine)
	if eng == "" {
		eng = "llama.cpp"
	}
	switch eng {
	// runtime flags mapping for more engines can be added here as needed
	case "llama.cpp":
		if cfg.Temperature != nil {
			flags = append(flags, "--temp", strconv.FormatFloat(*cfg.Temperature, 'f', -1, 64))
		}
		if cfg.TopP != nil {
			flags = append(flags, "--top-p", strconv.FormatFloat(*cfg.TopP, 'f', -1, 64))
		}
		if cfg.FrequencyPenalty != nil {
			flags = append(flags, "--frequency-penalty", strconv.FormatFloat(*cfg.FrequencyPenalty, 'f', -1, 64))
		}
		if cfg.PresencePenalty != nil {
			flags = append(flags, "--presence-penalty", strconv.FormatFloat(*cfg.PresencePenalty, 'f', -1, 64))
		}
		// Note: Context size already handled via --context-size during `docker model configure`
	default:
		// Unknown engine: no flags
	}
	return flags
}

// jsonSchema is a helper type that implements json.Marshaler for map[string]any
// This allows us to pass schema maps to the OpenAI library which expects json.Marshaler
type jsonSchema map[string]any

func (j jsonSchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any(j))
}
