package strategy

import (
	"context"

	"github.com/docker/cagent/pkg/rag/database"
)

// Strategy defines the interface for different retrieval strategies.
// This is the canonical definition used by both the strategies and rag packages.
type Strategy interface {
	// Initialize indexes all documents from the given paths.
	Initialize(ctx context.Context, docPaths []string, chunkSize, chunkOverlap int, respectWordBoundaries bool) error

	// Query searches for relevant documents using the strategy's retrieval method.
	// numResults is the maximum number of candidates to retrieve (before fusion).
	Query(ctx context.Context, query string, numResults int, threshold float64) ([]database.SearchResult, error)

	// CheckAndReindexChangedFiles checks for file changes and re-indexes if needed.
	CheckAndReindexChangedFiles(ctx context.Context, docPaths []string, chunkSize, chunkOverlap int, respectWordBoundaries bool) error

	// StartFileWatcher starts monitoring files for changes.
	StartFileWatcher(ctx context.Context, docPaths []string, chunkSize, chunkOverlap int, respectWordBoundaries bool) error

	// Close releases resources held by the strategy.
	Close() error
}

// Event represents a strategy lifecycle event. It is the canonical RAG event type
// used by strategies, the RAG manager, and the runtime.
type Event struct {
	Type         string
	StrategyName string // Name of the strategy emitting the event
	Message      string
	Progress     *Progress
	Error        error
	TotalTokens  int     // For usage events
	Cost         float64 // For usage events
}

// Progress represents progress within a multi-step operation (e.g., indexing).
type Progress struct {
	Current int
	Total   int
}

// Config contains a strategy and its runtime configuration.
type Config struct {
	Name                  string
	Strategy              Strategy
	Docs                  []string // Merged document paths (shared + strategy-specific)
	Limit                 int      // Max results for this strategy
	Threshold             float64  // Score threshold
	ChunkSize             int      // Chunk size for this strategy
	ChunkOverlap          int      // Chunk overlap for this strategy
	RespectWordBoundaries bool     // Whether to chunk on word boundaries
}
