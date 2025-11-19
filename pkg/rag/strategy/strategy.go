package strategy

import (
	"context"
	"fmt"

	v2 "github.com/docker/cagent/pkg/config/v2"
	"github.com/docker/cagent/pkg/environment"
)

// BuildContext contains everything needed to build a strategy
type BuildContext struct {
	RAGName       string
	ParentDir     string
	SharedDocs    []string
	Models        map[string]v2.ModelConfig
	Env           environment.Provider
	ModelsGateway string
}

// BuildStrategy builds a strategy from config
// Explicitly dispatches to the appropriate constructor based on type
func BuildStrategy(ctx context.Context, cfg v2.RAGStrategyConfig, buildCtx BuildContext, events chan<- Event) (*Config, error) {
	switch cfg.Type {
	case "chunked-embeddings":
		return NewChunkedEmbeddingsFromConfig(ctx, cfg, buildCtx, events)
	case "bm25":
		return NewBM25FromConfig(ctx, cfg, buildCtx, events)
	default:
		return nil, fmt.Errorf("unknown strategy type: %s (available: chunked-embeddings, bm25)", cfg.Type)
	}
}
