package strategy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	_ "modernc.org/sqlite"

	"github.com/docker/cagent/pkg/rag/database"
)

// VectorDatabase implements a SQLite database for vector embeddings storage
type VectorDatabase struct {
	db               *sql.DB
	vectorDimensions int
}

// NewVectorDatabase creates a new SQLite database for vector embeddings
func NewVectorDatabase(dbPath string, vectorDimensions int) (*VectorDatabase, error) {
	// Ensure parent directory exists
	if err := ensureDir(dbPath); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	vectorDB := &VectorDatabase{
		db:               db,
		vectorDimensions: vectorDimensions,
	}

	// Create schema
	if err := vectorDB.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	slog.Info("Vector database initialized",
		"vector_dimensions", vectorDimensions,
		"path", dbPath)

	return vectorDB, nil
}

// createSchema creates the database schema for vector embeddings
func (d *VectorDatabase) createSchema() error {
	schema := `
	-- File-level metadata (stored once per file, not per chunk)
	CREATE TABLE IF NOT EXISTS files (
		source_path TEXT PRIMARY KEY,
		file_hash TEXT NOT NULL,
		indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_file_hash ON files(file_hash);
	
	-- Chunk-level data with embeddings
	CREATE TABLE IF NOT EXISTS chunks (
		source_path TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		content TEXT NOT NULL,
		embedding BLOB NOT NULL,
		PRIMARY KEY (source_path, chunk_index),
		FOREIGN KEY (source_path) REFERENCES files(source_path) ON DELETE CASCADE
	);
	`

	_, err := d.db.Exec(schema)
	return err
}

// AddDocument adds or updates a document with its vector embedding
func (d *VectorDatabase) AddDocument(ctx context.Context, doc database.Document) error {
	// Validate vector dimensions
	if len(doc.Embedding) == 0 {
		return fmt.Errorf("embedding is required for vector database")
	}
	if len(doc.Embedding) != d.vectorDimensions {
		return fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(doc.Embedding), d.vectorDimensions)
	}

	embJSON, err := json.Marshal(doc.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Upsert file metadata (stores file_hash once, not on every chunk)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO files (source_path, file_hash, indexed_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(source_path) 
		 DO UPDATE SET file_hash = excluded.file_hash, indexed_at = CURRENT_TIMESTAMP`,
		doc.SourcePath, doc.FileHash)
	if err != nil {
		return fmt.Errorf("failed to upsert file metadata: %w", err)
	}

	// Upsert chunk data
	_, err = tx.ExecContext(ctx,
		`INSERT INTO chunks (source_path, chunk_index, content, embedding)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(source_path, chunk_index) 
		 DO UPDATE SET content = excluded.content, embedding = excluded.embedding`,
		doc.SourcePath, doc.ChunkIndex, doc.Content, embJSON)
	if err != nil {
		return fmt.Errorf("failed to upsert chunk: %w", err)
	}

	return tx.Commit()
}

// DeleteDocumentsByPath deletes all documents from a specific source file
func (d *VectorDatabase) DeleteDocumentsByPath(ctx context.Context, sourcePath string) error {
	// Delete file record (chunks cascade automatically via FK)
	_, err := d.db.ExecContext(ctx, "DELETE FROM files WHERE source_path = ?", sourcePath)
	return err
}

// SearchSimilar finds documents similar to the query embedding using cosine similarity
func (d *VectorDatabase) SearchSimilar(ctx context.Context, queryEmbedding []float64, limit int) ([]database.SearchResult, error) {
	// Join files and chunks - file metadata stored once, not duplicated per chunk
	query := `
	SELECT c.source_path, c.chunk_index, c.content, c.embedding, f.file_hash, f.indexed_at
	FROM chunks c
	JOIN files f ON c.source_path = f.source_path
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var results []database.SearchResult
	for rows.Next() {
		var doc database.Document
		var embJSON []byte

		if err := rows.Scan(&doc.SourcePath, &doc.ChunkIndex, &doc.Content,
			&embJSON, &doc.FileHash, &doc.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Generate document ID for compatibility
		doc.ID = fmt.Sprintf("%s_%d", doc.SourcePath, doc.ChunkIndex)

		var embedding []float64
		if err := json.Unmarshal(embJSON, &embedding); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
		}

		similarity := database.CosineSimilarity(queryEmbedding, embedding)
		results = append(results, database.SearchResult{
			Document:   doc,
			Similarity: similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Sort by similarity (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetDocumentsByPath retrieves all documents from a specific source file
func (d *VectorDatabase) GetDocumentsByPath(ctx context.Context, sourcePath string) ([]database.Document, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT c.source_path, c.chunk_index, c.content, f.file_hash, f.indexed_at
		 FROM chunks c
		 JOIN files f ON c.source_path = f.source_path
		 WHERE c.source_path = ?`,
		sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var docs []database.Document
	for rows.Next() {
		var doc database.Document
		if err := rows.Scan(&doc.SourcePath, &doc.ChunkIndex, &doc.Content,
			&doc.FileHash, &doc.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		// Generate document ID for compatibility
		doc.ID = fmt.Sprintf("%s_%d", doc.SourcePath, doc.ChunkIndex)
		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

// GetFileMetadata retrieves metadata for a specific source file
func (d *VectorDatabase) GetFileMetadata(ctx context.Context, sourcePath string) (*database.FileMetadata, error) {
	var metadata database.FileMetadata

	// Get file metadata and count chunks
	err := d.db.QueryRowContext(ctx,
		`SELECT f.source_path, f.file_hash, f.indexed_at, COUNT(c.chunk_index) as chunk_count
		 FROM files f
		 LEFT JOIN chunks c ON f.source_path = c.source_path
		 WHERE f.source_path = ?
		 GROUP BY f.source_path, f.file_hash, f.indexed_at`,
		sourcePath).Scan(&metadata.SourcePath, &metadata.FileHash, &metadata.LastIndexed, &metadata.ChunkCount)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	return &metadata, nil
}

// SetFileMetadata stores or updates metadata for a source file
func (d *VectorDatabase) SetFileMetadata(ctx context.Context, metadata database.FileMetadata) error {
	// Update file metadata
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO files (source_path, file_hash, indexed_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(source_path) 
		 DO UPDATE SET file_hash = excluded.file_hash, indexed_at = CURRENT_TIMESTAMP`,
		metadata.SourcePath, metadata.FileHash)
	return err
}

// GetAllFileMetadata retrieves metadata for all indexed files
func (d *VectorDatabase) GetAllFileMetadata(ctx context.Context) ([]database.FileMetadata, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT f.source_path, f.file_hash, f.indexed_at, COUNT(c.chunk_index) as chunk_count
		 FROM files f
		 LEFT JOIN chunks c ON f.source_path = c.source_path
		 GROUP BY f.source_path, f.file_hash, f.indexed_at`)
	if err != nil {
		return nil, fmt.Errorf("failed to query file metadata: %w", err)
	}
	defer rows.Close()

	var metadata []database.FileMetadata
	for rows.Next() {
		var m database.FileMetadata
		if err := rows.Scan(&m.SourcePath, &m.FileHash, &m.LastIndexed, &m.ChunkCount); err != nil {
			return nil, fmt.Errorf("failed to scan metadata row: %w", err)
		}
		metadata = append(metadata, m)
	}

	return metadata, rows.Err()
}

// DeleteFileMetadata deletes metadata for a specific source file
func (d *VectorDatabase) DeleteFileMetadata(ctx context.Context, sourcePath string) error {
	// Delete file record (chunks cascade automatically via FK)
	_, err := d.db.ExecContext(ctx, "DELETE FROM files WHERE source_path = ?", sourcePath)
	return err
}

// Close closes the database connection
func (d *VectorDatabase) Close() error {
	// Checkpoint WAL to merge changes into main database file
	// This reduces .db-wal file size and ensures clean shutdown
	if _, err := d.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		slog.Warn("Failed to checkpoint WAL before close", "error", err)
	}

	return d.db.Close()
}
