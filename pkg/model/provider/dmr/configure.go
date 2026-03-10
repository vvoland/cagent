package dmr

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/docker/docker-agent/pkg/config/latest"
)

// configureRequest mirrors the model-runner's scheduling.ConfigureRequest structure.
// It specifies per-model runtime configuration options sent via POST /engines/_configure.
type configureRequest struct {
	Model        string                      `json:"model"`
	ContextSize  *int32                      `json:"context-size,omitempty"`
	RuntimeFlags []string                    `json:"runtime-flags,omitempty"`
	Speculative  *speculativeDecodingRequest `json:"speculative,omitempty"`
}

// speculativeDecodingRequest mirrors model-runner's inference.SpeculativeDecodingConfig.
type speculativeDecodingRequest struct {
	DraftModel        string  `json:"draft_model,omitempty"`
	NumTokens         int     `json:"num_tokens,omitempty"`
	MinAcceptanceRate float64 `json:"min_acceptance_rate,omitempty"`
}

type speculativeDecodingOpts struct {
	draftModel     string
	numTokens      int
	acceptanceRate float64
}

// configureModel sends model configuration to Model Runner via POST /engines/_configure.
func configureModel(ctx context.Context, httpClient *http.Client, baseURL, model string, contextSize *int64, runtimeFlags []string, specOpts *speculativeDecodingOpts) error {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	configureURL := buildConfigureURL(baseURL)
	reqData, err := json.Marshal(buildConfigureRequest(model, contextSize, runtimeFlags, specOpts))
	if err != nil {
		return fmt.Errorf("failed to marshal configure request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, configureTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, configureURL, bytes.NewReader(reqData))
	if err != nil {
		return fmt.Errorf("failed to create configure request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("Sending model configure request",
		"model", model,
		"url", configureURL,
		"context_size", contextSize,
		"runtime_flags", runtimeFlags,
		"speculative_opts", specOpts)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("configure request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("configure request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	slog.Debug("Model configure completed", "model", model)
	return nil
}

// buildConfigureURL derives the /engines/_configure endpoint URL from the OpenAI base URL.
// It handles various URL formats:
//   - http://host:port/engines/v1/ → http://host:port/engines/_configure
//   - http://_/exp/vDD4.40/engines/v1 → http://_/exp/vDD4.40/engines/_configure
//   - http://host:port/engines/llama.cpp/v1/ → http://host:port/engines/llama.cpp/_configure
func buildConfigureURL(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return strings.TrimSuffix(strings.TrimSuffix(baseURL, "/"), "/v1") + "/_configure"
	}

	path := strings.TrimSuffix(strings.TrimSuffix(u.Path, "/"), "/v1")
	u.Path = path + "/_configure"
	return u.String()
}

// buildConfigureRequest constructs the JSON request body for POST /engines/_configure.
func buildConfigureRequest(model string, contextSize *int64, runtimeFlags []string, specOpts *speculativeDecodingOpts) configureRequest {
	req := configureRequest{
		Model:        model,
		RuntimeFlags: runtimeFlags,
	}

	if contextSize != nil {
		cs := int32(*contextSize)
		req.ContextSize = &cs
	}

	if specOpts != nil {
		req.Speculative = &speculativeDecodingRequest{
			DraftModel:        specOpts.draftModel,
			NumTokens:         specOpts.numTokens,
			MinAcceptanceRate: specOpts.acceptanceRate,
		}
	}

	return req
}

// mergeRuntimeFlagsPreferUser merges derived engine flags (from model config fields like
// `temperature`) and user-provided runtime flags (from `provider_opts.runtime_flags`).
// When both specify the same flag key (e.g. --temp), the user value wins and a warning
// is returned. Order: non-conflicting derived flags first, then all user flags.
func mergeRuntimeFlagsPreferUser(derived, user []string) (merged, warnings []string) {
	// parsedFlag holds a parsed flag token (e.g. "--temp 0.5" → key="--temp", tokens=["--temp","0.5"]).
	type parsedFlag struct {
		key    string
		tokens []string
	}

	parse := func(args []string) []parsedFlag {
		var out []parsedFlag
		for i := 0; i < len(args); i++ {
			tok := args[i]
			if !strings.HasPrefix(tok, "-") {
				out = append(out, parsedFlag{key: tok, tokens: []string{tok}})
				continue
			}
			// --key=value
			if k, _, found := strings.Cut(tok, "="); found {
				out = append(out, parsedFlag{key: k, tokens: []string{tok}})
				continue
			}
			// --key value (next token is the value if it doesn't start with -)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				out = append(out, parsedFlag{key: tok, tokens: []string{tok, args[i+1]}})
				i++
			} else {
				out = append(out, parsedFlag{key: tok, tokens: []string{tok}})
			}
		}
		return out
	}

	derFlags := parse(derived)
	usrFlags := parse(user)

	// Build a set of flag keys the user explicitly provides.
	userKeys := make(map[string]bool, len(usrFlags))
	for _, f := range usrFlags {
		if strings.HasPrefix(f.key, "-") {
			userKeys[f.key] = true
		}
	}

	// Emit non-conflicting derived flags; warn on conflicts.
	for _, f := range derFlags {
		if strings.HasPrefix(f.key, "-") && userKeys[f.key] {
			warnings = append(warnings, "Overriding runtime flag "+f.key+" with value from provider_opts.runtime_flags")
			continue
		}
		merged = append(merged, f.tokens...)
	}
	for _, f := range usrFlags {
		merged = append(merged, f.tokens...)
	}
	return merged, warnings
}

// buildRuntimeFlagsFromModelConfig converts standard ModelConfig fields into backend-specific
// runtime flags that the model-runner understands when launching the engine.
// Currently supports "llama.cpp". Unknown engines produce no flags.
func buildRuntimeFlagsFromModelConfig(engine string, cfg *latest.ModelConfig) []string {
	if cfg == nil {
		return nil
	}

	eng := cmp.Or(strings.TrimSpace(engine), "llama.cpp")
	if eng != "llama.cpp" {
		return nil
	}

	var flags []string
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
	return flags
}

// parseFloat64 attempts to parse a value as float64 from various types.
func parseFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint64:
		return float64(t), true
	case string:
		if s := strings.TrimSpace(t); s != "" {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

// parseInt attempts to parse a value as int from various types.
func parseInt(v any) (int, bool) {
	if f, ok := parseFloat64(v); ok {
		return int(f), true
	}
	return 0, false
}

// parseDMRProviderOpts extracts DMR-specific provider options from the model config:
// context size, runtime flags, and speculative decoding settings.
func parseDMRProviderOpts(cfg *latest.ModelConfig) (contextSize *int64, runtimeFlags []string, specOpts *speculativeDecodingOpts) {
	if cfg == nil {
		return nil, nil, nil
	}

	contextSize = cfg.MaxTokens

	slog.Debug("DMR provider opts", "provider_opts", cfg.ProviderOpts)

	if len(cfg.ProviderOpts) == 0 {
		return contextSize, nil, nil
	}

	runtimeFlags = parseRuntimeFlags(cfg.ProviderOpts)
	specOpts = parseSpeculativeOpts(cfg.ProviderOpts)

	return contextSize, runtimeFlags, specOpts
}

// parseRuntimeFlags extracts the "runtime_flags" key from provider opts.
func parseRuntimeFlags(opts map[string]any) []string {
	v, ok := opts["runtime_flags"]
	if !ok {
		return nil
	}

	switch t := v.(type) {
	case []any:
		flags := make([]string, 0, len(t))
		for _, item := range t {
			flags = append(flags, fmt.Sprint(item))
		}
		return flags
	case []string:
		return append([]string(nil), t...)
	case string:
		return strings.Fields(strings.ReplaceAll(t, ",", " "))
	default:
		return nil
	}
}

// parseSpeculativeOpts extracts speculative decoding options from provider opts.
func parseSpeculativeOpts(opts map[string]any) *speculativeDecodingOpts {
	var so speculativeDecodingOpts
	var found bool

	if v, ok := opts["speculative_draft_model"]; ok {
		if s, ok := v.(string); ok && s != "" {
			so.draftModel = s
			found = true
		}
	}
	if v, ok := opts["speculative_num_tokens"]; ok {
		if n, ok := parseInt(v); ok {
			so.numTokens = n
			found = true
		}
	}
	if v, ok := opts["speculative_acceptance_rate"]; ok {
		if f, ok := parseFloat64(v); ok {
			so.acceptanceRate = f
			found = true
		}
	}

	if !found {
		return nil
	}
	return &so
}
