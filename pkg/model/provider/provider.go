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
	"github.com/docker/cagent/pkg/model/provider/options"
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

func New(cfg *config.ModelConfig, env environment.Provider, logger *slog.Logger, opts ...options.Opt) (Provider, error) {
	logger.Debug("Creating model provider", "type", cfg.Provider, "model", cfg.Model)

	switch cfg.Provider {
	case "openai":
		return openai.NewClient(cfg, env, logger, opts...)

	case "anthropic":
		return anthropic.NewClient(cfg, env, logger, opts...)

	case "dmr":
		return dmr.NewClient(cfg, logger, opts...)

	default:
		logger.Error("Unknown provider type", "type", cfg.Provider)
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Provider)
	}
}
