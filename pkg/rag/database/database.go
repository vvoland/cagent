package database

import (
	"context"
	"math"
)

// Document represents a chunk of text with its embedding
type Document struct {
	ID         string    `json:"id"`
	SourcePath string    `json:"source_path"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content"`
	Embedding  []float64 `json:"-"`
	FileHash   string    `json:"file_hash"`
	CreatedAt  string    `json:"created_at"`
}

// SearchResult represents a document with its similarity score
type SearchResult struct {
	Document   Document `json:"document"`
	Similarity float64  `json:"similarity"`
}

// FileMetadata represents metadata about an indexed file
type FileMetadata struct {
	SourcePath  string
	FileHash    string
	LastIndexed string
	ChunkCount  int
}

// Database interface for RAG operations
// Implementations: SQLite (sqlite.go), PostgreSQL (future), Pinecone (future), etc.
type Database interface {
	// Document operations
	AddDocument(ctx context.Context, doc Document) error
	DeleteDocumentsByPath(ctx context.Context, sourcePath string) error
	SearchSimilar(ctx context.Context, queryEmbedding []float64, limit int) ([]SearchResult, error)
	GetDocumentsByPath(ctx context.Context, sourcePath string) ([]Document, error)

	// File metadata operations (for change detection and incremental indexing)
	GetFileMetadata(ctx context.Context, sourcePath string) (*FileMetadata, error)
	SetFileMetadata(ctx context.Context, metadata FileMetadata) error
	GetAllFileMetadata(ctx context.Context) ([]FileMetadata, error)
	DeleteFileMetadata(ctx context.Context, sourcePath string) error

	// Resource management
	Close() error
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

// SortByScore sorts results by similarity in descending order
func SortByScore(results []SearchResult) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
