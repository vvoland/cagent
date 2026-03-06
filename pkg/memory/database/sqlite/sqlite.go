package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/docker/cagent/pkg/memory/database"
	"github.com/docker/cagent/pkg/sqliteutil"
)

type MemoryDatabase struct {
	db *sql.DB
}

func NewMemoryDatabase(path string) (database.Database, error) {
	db, err := sqliteutil.OpenDB(path)
	if err != nil {
		return nil, err
	}
	// Ensure we close the connection if table creation fails
	// Note: We don't defer close here because we return the db on success

	_, err = db.ExecContext(context.Background(), "CREATE TABLE IF NOT EXISTS memories (id TEXT PRIMARY KEY, created_at TEXT, memory TEXT)")
	if err != nil {
		db.Close()
		return nil, err
	}

	// Add category column if it doesn't exist (transparent migration)
	_, _ = db.ExecContext(context.Background(), "ALTER TABLE memories ADD COLUMN category TEXT DEFAULT ''")

	return &MemoryDatabase{db: db}, nil
}

func (m *MemoryDatabase) AddMemory(ctx context.Context, memory database.UserMemory) error {
	if memory.ID == "" {
		return database.ErrEmptyID
	}
	_, err := m.db.ExecContext(ctx, "INSERT INTO memories (id, created_at, memory, category) VALUES (?, ?, ?, ?)",
		memory.ID, memory.CreatedAt, memory.Memory, memory.Category)
	return err
}

func (m *MemoryDatabase) GetMemories(ctx context.Context) ([]database.UserMemory, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT id, created_at, memory, COALESCE(category, '') FROM memories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []database.UserMemory
	for rows.Next() {
		var memory database.UserMemory
		err := rows.Scan(&memory.ID, &memory.CreatedAt, &memory.Memory, &memory.Category)
		if err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memories, nil
}

func (m *MemoryDatabase) DeleteMemory(ctx context.Context, memory database.UserMemory) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM memories WHERE id = ?", memory.ID)
	return err
}

func (m *MemoryDatabase) SearchMemories(ctx context.Context, query string, category string) ([]database.UserMemory, error) {
	var conditions []string
	var args []any

	if query != "" {
		words := strings.Fields(query)
		for _, word := range words {
			conditions = append(conditions, "LOWER(memory) LIKE LOWER(?)")
			args = append(args, "%"+word+"%")
		}
	}

	if category != "" {
		conditions = append(conditions, "LOWER(category) = LOWER(?)")
		args = append(args, category)
	}

	sql := "SELECT id, created_at, memory, COALESCE(category, '') FROM memories"
	if len(conditions) > 0 {
		sql += " WHERE " + strings.Join(conditions, " AND ")
	}

	rows, err := m.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []database.UserMemory
	for rows.Next() {
		var memory database.UserMemory
		err := rows.Scan(&memory.ID, &memory.CreatedAt, &memory.Memory, &memory.Category)
		if err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memories, nil
}

func (m *MemoryDatabase) UpdateMemory(ctx context.Context, memory database.UserMemory) error {
	if memory.ID == "" {
		return database.ErrEmptyID
	}

	result, err := m.db.ExecContext(ctx, "UPDATE memories SET memory = ?, category = ? WHERE id = ?",
		memory.Memory, memory.Category, memory.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("%w: %s", database.ErrMemoryNotFound, memory.ID)
	}

	return nil
}
