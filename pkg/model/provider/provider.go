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

func New(cfg *config.ModelConfig, env environment.Provider, logger *slog.Logger) (Provider, error) {
	logger.Debug("Creating model provider", "type", cfg.Type, "model", cfg.Model)

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
