package strategy

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/docker/cagent/pkg/rag/database"
)

// BM25Database implements a simple database for BM25 strategy (no vectors needed)
type BM25Database struct {
	db *sql.DB
}

// NewBM25Database creates a new SQLite database for BM25 strategy
func NewBM25Database(dbPath string) (*BM25Database, error) {
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

	bm25DB := &BM25Database{db: db}

	// Create schema
	if err := bm25DB.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	slog.Info("BM25 database initialized", "path", dbPath)

	return bm25DB, nil
}

// createSchema creates the simple schema for BM25 (no vectors)
func (d *BM25Database) createSchema() error {
	schema := `
	-- Document metadata (no vectors needed for BM25)
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		source_path TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		content TEXT NOT NULL,
		file_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(source_path, chunk_index)
	);
	CREATE INDEX IF NOT EXISTS idx_source_path ON documents(source_path);
	CREATE INDEX IF NOT EXISTS idx_file_hash ON documents(file_hash);
	CREATE INDEX IF NOT EXISTS idx_content_fts ON documents(content);
	
	-- File metadata for incremental indexing
	CREATE TABLE IF NOT EXISTS file_metadata (
		source_path TEXT PRIMARY KEY,
		file_hash TEXT NOT NULL,
		last_indexed DATETIME DEFAULT CURRENT_TIMESTAMP,
		chunk_count INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_metadata_hash ON file_metadata(file_hash);
	`

	_, err := d.db.Exec(schema)
	return err
}

// AddDocument adds or updates a document (no embedding needed)
func (d *BM25Database) AddDocument(ctx context.Context, doc database.Document) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Check if document exists
	var exists bool
	err = tx.QueryRowContext(ctx,
		"SELECT 1 FROM documents WHERE source_path = ? AND chunk_index = ?",
		doc.SourcePath, doc.ChunkIndex).Scan(&exists)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing document: %w", err)
	}

	if exists {
		// Update existing
		_, err = tx.ExecContext(ctx,
			`UPDATE documents SET content = ?, file_hash = ?, created_at = CURRENT_TIMESTAMP 
			 WHERE source_path = ? AND chunk_index = ?`,
			doc.Content, doc.FileHash, doc.SourcePath, doc.ChunkIndex)
		if err != nil {
			return fmt.Errorf("failed to update document: %w", err)
		}
	} else {
		// Insert new
		_, err = tx.ExecContext(ctx,
			`INSERT INTO documents (id, source_path, chunk_index, content, file_hash)
			 VALUES (?, ?, ?, ?, ?)`,
			doc.ID, doc.SourcePath, doc.ChunkIndex, doc.Content, doc.FileHash)
		if err != nil {
			return fmt.Errorf("failed to insert document: %w", err)
		}
	}

	return tx.Commit()
}

// DeleteDocumentsByPath deletes all documents from a specific source file
func (d *BM25Database) DeleteDocumentsByPath(ctx context.Context, sourcePath string) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM documents WHERE source_path = ?", sourcePath)
	return err
}

// SearchSimilar is not used by BM25 (it does its own scoring)
// This just returns all documents for BM25 to score
func (d *BM25Database) SearchSimilar(ctx context.Context, _ []float64, limit int) ([]database.SearchResult, error) {
	query := `
	SELECT id, source_path, chunk_index, content, file_hash, created_at
	FROM documents
	LIMIT ?
	`

	rows, err := d.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var results []database.SearchResult
	for rows.Next() {
		var doc database.Document
		if err := rows.Scan(&doc.ID, &doc.SourcePath, &doc.ChunkIndex, &doc.Content,
			&doc.FileHash, &doc.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = append(results, database.SearchResult{
			Document:   doc,
			Similarity: 0, // BM25 calculates its own scores
		})
	}

	return results, rows.Err()
}

// GetDocumentsByPath retrieves all documents from a specific source file
func (d *BM25Database) GetDocumentsByPath(ctx context.Context, sourcePath string) ([]database.Document, error) {
	rows, err := d.db.QueryContext(ctx,
		"SELECT id, source_path, chunk_index, content, file_hash, created_at FROM documents WHERE source_path = ?",
		sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var docs []database.Document
	for rows.Next() {
		var doc database.Document
		if err := rows.Scan(&doc.ID, &doc.SourcePath, &doc.ChunkIndex, &doc.Content,
			&doc.FileHash, &doc.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

// GetFileMetadata retrieves metadata for a specific source file
func (d *BM25Database) GetFileMetadata(ctx context.Context, sourcePath string) (*database.FileMetadata, error) {
	var metadata database.FileMetadata
	err := d.db.QueryRowContext(ctx,
		"SELECT source_path, file_hash, last_indexed, chunk_count FROM file_metadata WHERE source_path = ?",
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
func (d *BM25Database) SetFileMetadata(ctx context.Context, metadata database.FileMetadata) error {
	query := `
	INSERT INTO file_metadata (source_path, file_hash, last_indexed, chunk_count)
	VALUES (?, ?, CURRENT_TIMESTAMP, ?)
	ON CONFLICT(source_path) DO UPDATE SET
		file_hash = excluded.file_hash,
		last_indexed = CURRENT_TIMESTAMP,
		chunk_count = excluded.chunk_count
	`

	_, err := d.db.ExecContext(ctx, query, metadata.SourcePath, metadata.FileHash, metadata.ChunkCount)
	return err
}

// GetAllFileMetadata retrieves metadata for all indexed files
func (d *BM25Database) GetAllFileMetadata(ctx context.Context) ([]database.FileMetadata, error) {
	rows, err := d.db.QueryContext(ctx,
		"SELECT source_path, file_hash, last_indexed, chunk_count FROM file_metadata")
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
func (d *BM25Database) DeleteFileMetadata(ctx context.Context, sourcePath string) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM file_metadata WHERE source_path = ?", sourcePath)
	return err
}

// Close closes the database connection
func (d *BM25Database) Close() error {
	// Checkpoint WAL to merge changes into main database file
	// This reduces .db-wal file size and ensures clean shutdown
	if _, err := d.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		slog.Warn("Failed to checkpoint WAL before close", "error", err)
	}

	return d.db.Close()
}

// ensureDir creates the parent directory for a file path if it doesn't exist
func ensureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir == "" || dir == "." {
		return nil
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o755)
	}

	return nil
}
