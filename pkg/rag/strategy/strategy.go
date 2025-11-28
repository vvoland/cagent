package strategy

import (
	"context"
	"fmt"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/rag/types"
)

// BuildContext contains everything needed to build a strategy
type BuildContext struct {
	RAGName       string
	ParentDir     string
	SharedDocs    []string
	Models        map[string]latest.ModelConfig
	Env           environment.Provider
	ModelsGateway string
	RespectVCS    bool // Whether to respect VCS ignore files (e.g., .gitignore) when collecting files
}

// BuildStrategy builds a strategy from config
// Explicitly dispatches to the appropriate constructor based on type
func BuildStrategy(ctx context.Context, cfg latest.RAGStrategyConfig, buildCtx BuildContext, events chan<- types.Event) (*Config, error) {
	switch cfg.Type {
	case "chunked-embeddings":
		return NewChunkedEmbeddingsFromConfig(ctx, cfg, buildCtx, events)
	case "semantic-embeddings":
		return NewSemanticEmbeddingsFromConfig(ctx, cfg, buildCtx, events)
	case "bm25":
		return NewBM25FromConfig(ctx, cfg, buildCtx, events)
	default:
		return nil, fmt.Errorf("unknown strategy type: %s (available: chunked-embeddings, semantic-embeddings, bm25)", cfg.Type)
	}
}
