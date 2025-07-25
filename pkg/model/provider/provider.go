package provider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider/anthropic"
	"github.com/docker/cagent/pkg/model/provider/dmr"
	"github.com/docker/cagent/pkg/model/provider/openai"
	"github.com/docker/cagent/pkg/tools"
)

// Provider defines the interface for model providers
type Provider interface {
	// CreateChatCompletionStream creates a streaming chat completion request
	// It returns a stream that can be iterated over to get completion chunks
	CreateChatCompletionStream(
		ctx context.Context,
		messages []chat.Message,
		tools []tools.Tool,
	) (chat.MessageStream, error)

	CreateChatCompletion(
		ctx context.Context,
		messages []chat.Message,
	) (string, error)
}

func New(cfg *config.ModelConfig, env environment.Provider, gateway string, logger *slog.Logger) (Provider, error) {
	logger.Debug("Creating model provider", "type", cfg.Type, "model", cfg.Model)

	if gateway != "" {
		api_key, err := env.Get(context.TODO(), "LITELLM_API_KEY")
		if err != nil {
			return nil, err
		}

		env = environment.NewMultiProvider(
			environment.NewKeyValueProvider(map[string]string{
				"OPENAI_API_KEY": api_key,
			}),
			env,
		)

		gatewayCfg := &config.ModelConfig{
			Type:              "openai",
			Model:             cfg.Model,
			BaseURL:           strings.TrimSuffix(gateway, "/") + "/v1",
			ParallelToolCalls: cfg.ParallelToolCalls,
			// TODO(dga): temperature and stuff.
			Temperature:      cfg.Temperature,
			MaxTokens:        cfg.MaxTokens,
			TopP:             cfg.TopP,
			FrequencyPenalty: cfg.FrequencyPenalty,
			PresencePenalty:  cfg.PresencePenalty,
		}

		return openai.NewClient(gatewayCfg, env, logger)
	}

	switch cfg.Type {
	case "openai":
		return openai.NewClient(cfg, env, logger)
	case "anthropic":
		return anthropic.NewClient(cfg, env, logger)
	case "dmr":
		return dmr.NewClient(cfg, logger)
	}

	logger.Error("Unknown provider type", "type", cfg.Type)
	return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
}
