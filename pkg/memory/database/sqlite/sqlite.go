package sqlite

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rumpl/cagent/pkg/memory/database"
)

type SqliteMemoryDatabase struct {
	db *sql.DB
}

func NewSqliteMemoryDatabase(path string) (database.Database, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.ExecContext(context.Background(), "CREATE TABLE IF NOT EXISTS memories (id TEXT PRIMARY KEY, created_at TEXT, memory TEXT)")
	if err != nil {
		return nil, err
	}

	return &SqliteMemoryDatabase{db: db}, nil
}

func (m *SqliteMemoryDatabase) AddMemory(ctx context.Context, memory database.UserMemory) error {
	_, err := m.db.ExecContext(ctx, "INSERT INTO memories (id, created_at, memory) VALUES (?, ?, ?)",
		memory.ID, memory.CreatedAt, memory.Memory)
	return err
}

func (m *SqliteMemoryDatabase) GetMemories(ctx context.Context) ([]database.UserMemory, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT id, created_at, memory FROM memories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []database.UserMemory
	for rows.Next() {
		var memory database.UserMemory
		err := rows.Scan(&memory.ID, &memory.CreatedAt, &memory.Memory)
		if err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

func (m *SqliteMemoryDatabase) DeleteMemory(ctx context.Context, memory database.UserMemory) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM memories WHERE id = ?", memory.ID)
	return err
}
