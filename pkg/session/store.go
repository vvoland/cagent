package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrEmptyID  = errors.New("session ID cannot be empty")
	ErrNotFound = errors.New("session not found")
)

// convertMessagesToItems converts a slice of Messages to SessionItems for backward compatibility
func convertMessagesToItems(messages []Message) []Item {
	items := make([]Item, len(messages))
	for i := range messages {
		items[i] = NewMessageItem(&messages[i])
	}
	return items
}

// Store defines the interface for session storage
type Store interface {
	AddSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	GetSessions(ctx context.Context) ([]*Session, error)
	DeleteSession(ctx context.Context, id string) error
	UpdateSession(ctx context.Context, session *Session) error
}

// SQLiteSessionStore implements Store using SQLite
type SQLiteSessionStore struct {
	db *sql.DB
}

// NewSQLiteSessionStore creates a new SQLite session store
func NewSQLiteSessionStore(path string) (Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			messages TEXT,
			created_at TEXT
		)
	`)
	if err != nil {
		return nil, err
	}

	// Initialize and run migrations
	migrationManager := NewMigrationManager(db)
	err = migrationManager.InitializeMigrations(context.Background())
	if err != nil {
		return nil, err
	}

	return &SQLiteSessionStore{db: db}, nil
}

// AddSession adds a new session to the store
func (s *SQLiteSessionStore) AddSession(ctx context.Context, session *Session) error {
	if session.ID == "" {
		return ErrEmptyID
	}

	itemsJSON, err := json.Marshal(session.Messages)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO sessions (id, messages, tools_approved, created_at) VALUES (?, ?, ?, ?)",
		session.ID, string(itemsJSON), session.ToolsApproved, session.CreatedAt.Format(time.RFC3339))
	return err
}

// GetSession retrieves a session by ID
func (s *SQLiteSessionStore) GetSession(ctx context.Context, id string) (*Session, error) {
	if id == "" {
		return nil, ErrEmptyID
	}

	row := s.db.QueryRowContext(ctx,
		"SELECT id, messages, tools_approved, created_at FROM sessions WHERE id = ?", id)

	var messagesJSON, toolsApprovedStr, createdAtStr string
	var sessionID string

	err := row.Scan(&sessionID, &messagesJSON, &toolsApprovedStr, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Ok listen up, we used to only store messages in the database, but now we
	// store messages and sub-sessions. So we need to handle both cases.
	// We do this in a kind of hacky way, but it works. "AgentFilename" is always present
	// in a message in the old format, so we check for it to determine the format.
	var items []Item
	var messages []Message
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return nil, err
	}
	if len(messages) > 0 && messages[0].AgentFilename == "" {
		if err := json.Unmarshal([]byte(messagesJSON), &items); err != nil {
			return nil, err
		}
	} else {
		items = convertMessagesToItems(messages)
	}

	toolsApproved, err := strconv.ParseBool(toolsApprovedStr)
	if err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:            sessionID,
		Messages:      items,
		ToolsApproved: toolsApproved,
		CreatedAt:     createdAt,
		logger:        nil, // Logger is not persisted and will need to be set by caller
	}, nil
}

// GetSessions retrieves all sessions
func (s *SQLiteSessionStore) GetSessions(ctx context.Context) ([]*Session, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, messages, tools_approved, created_at FROM sessions ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		var messagesJSON, toolsApprovedStr, createdAtStr string
		var sessionID string

		err := rows.Scan(&sessionID, &messagesJSON, &toolsApprovedStr, &createdAtStr)
		if err != nil {
			return nil, err
		}

		// Ok listen up, we used to only store messages in the database, but now we
		// store messages and sub-sessions. So we need to handle both cases.
		// We do this in a kind of hacky way, but it works. "AgentFilename" is always present
		// in a message in the old format, so we check for it to determine the format.
		var items []Item
		var messages []Message
		if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
			return nil, err
		}
		if len(messages) > 0 && messages[0].AgentFilename == "" {
			if err := json.Unmarshal([]byte(messagesJSON), &items); err != nil {
				return nil, err
			}
		} else {
			items = convertMessagesToItems(messages)
		}

		toolsApproved, err := strconv.ParseBool(toolsApprovedStr)
		if err != nil {
			return nil, err
		}

		createdAt, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, err
		}

		session := &Session{
			ID:            sessionID,
			Messages:      items,
			ToolsApproved: toolsApproved,
			CreatedAt:     createdAt,
			logger:        nil, // Logger is not persisted and will need to be set by caller
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// DeleteSession deletes a session by ID
func (s *SQLiteSessionStore) DeleteSession(ctx context.Context, id string) error {
	if id == "" {
		return ErrEmptyID
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateSession updates an existing session
func (s *SQLiteSessionStore) UpdateSession(ctx context.Context, session *Session) error {
	if session.ID == "" {
		return ErrEmptyID
	}

	itemsJSON, err := json.Marshal(session.Messages)
	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE sessions SET messages = ?, tools_approved = ? WHERE id = ?",
		string(itemsJSON), session.ToolsApproved, session.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteSessionStore) Close() error {
	return s.db.Close()
}
