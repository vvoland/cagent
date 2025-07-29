package provider

import (
	"context"
	"fmt"
	"log/slog"

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

	switch {
	case gateway != "":
		gatewayCfg := &config.ModelConfig{
			Type:              "openai",
			Model:             cfg.Model,
			BaseURL:           gateway + "/v1",
			ParallelToolCalls: cfg.ParallelToolCalls,
			// MaxTokens:        cfg.MaxTokens, // MaxTokens is not portable
			// TODO(dga): temperature and stuff.
		}

		return openai.NewClient(gatewayCfg, env, logger)

	case cfg.Type == "openai":
		return openai.NewClient(cfg, env, logger)

	case cfg.Type == "anthropic":
		return anthropic.NewClient(cfg, env, logger)

	case cfg.Type == "dmr":
		return dmr.NewClient(cfg, logger)

	default:
		logger.Error("Unknown provider type", "type", cfg.Type)
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}
