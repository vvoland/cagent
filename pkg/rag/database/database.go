package database

import (
	"math"
)

// Document represents a chunk of text - the base type returned by all RAG strategies.
// Strategy-specific fields (embeddings, semantic summaries) are handled internally
// by each strategy and don't need to be exposed here.
type Document struct {
	ID         string `json:"id"`
	SourcePath string `json:"source_path"`
	ChunkIndex int    `json:"chunk_index"`
	Content    string `json:"content"`
	FileHash   string `json:"file_hash"`
	CreatedAt  string `json:"created_at"`
}

// SearchResult represents a document with its relevance score.
// This is the common return type for all Strategy.Query() implementations.
type SearchResult struct {
	Document   Document `json:"document"`
	Similarity float64  `json:"similarity"`
}

// FileMetadata represents metadata about an indexed file.
// Used for change detection and incremental indexing.
type FileMetadata struct {
	SourcePath  string
	FileHash    string
	LastIndexed string
	ChunkCount  int
}

// Helper functions

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
