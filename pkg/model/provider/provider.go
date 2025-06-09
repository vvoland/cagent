package provider

import (
	"context"
	"fmt"

	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/model/provider/anthropic"
	"github.com/rumpl/cagent/pkg/model/provider/dmr"
	"github.com/rumpl/cagent/pkg/model/provider/openai"
	"github.com/rumpl/cagent/pkg/tools"
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
}

// Factory interface for creating model providers
type Factory interface {
	// NewProvider creates a new provider from a model config
	NewProvider(cfg *config.ModelConfig) (Provider, error)
}

type factory struct{}

func NewFactory() Factory {
	return &factory{}
}

func (f *factory) NewProvider(cfg *config.ModelConfig) (Provider, error) {
	switch cfg.Type {
	case "openai":
		return openai.NewClient(cfg)
	case "anthropic":
		return anthropic.NewClient(cfg)
	case "dmr":
		return dmr.NewClient(cfg)
	}
	return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
}
