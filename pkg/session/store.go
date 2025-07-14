package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrEmptyID  = errors.New("session ID cannot be empty")
	ErrNotFound = errors.New("session not found")
)

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

	// Create the sessions table if it doesn't exist
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			messages TEXT,
			state TEXT,
			created_at TEXT
		)
	`)
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

	messagesJSON, err := json.Marshal(session.Messages)
	if err != nil {
		return err
	}

	stateJSON, err := json.Marshal(session.State)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO sessions (id, messages, state, created_at) VALUES (?, ?, ?, ?)",
		session.ID, string(messagesJSON), string(stateJSON), session.CreatedAt.Format(time.RFC3339))
	return err
}

// GetSession retrieves a session by ID
func (s *SQLiteSessionStore) GetSession(ctx context.Context, id string) (*Session, error) {
	if id == "" {
		return nil, ErrEmptyID
	}

	row := s.db.QueryRowContext(ctx,
		"SELECT id, messages, state, created_at FROM sessions WHERE id = ?", id)

	var messagesJSON, stateJSON, createdAtStr string
	var sessionID string

	err := row.Scan(&sessionID, &messagesJSON, &stateJSON, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Parse the data
	var messages []AgentMessage
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return nil, err
	}

	var state map[string]any
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        sessionID,
		Messages:  messages,
		State:     state,
		CreatedAt: createdAt,
		logger:    nil, // Logger is not persisted and will need to be set by caller
	}, nil
}

// GetSessions retrieves all sessions
func (s *SQLiteSessionStore) GetSessions(ctx context.Context) ([]*Session, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, messages, state, created_at FROM sessions ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		var messagesJSON, stateJSON, createdAtStr string
		var sessionID string

		err := rows.Scan(&sessionID, &messagesJSON, &stateJSON, &createdAtStr)
		if err != nil {
			return nil, err
		}

		// Parse the data
		var messages []AgentMessage
		if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
			return nil, err
		}

		var state map[string]any
		if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
			return nil, err
		}

		createdAt, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, err
		}

		session := &Session{
			ID:        sessionID,
			Messages:  messages,
			State:     state,
			CreatedAt: createdAt,
			logger:    nil, // Logger is not persisted and will need to be set by caller
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

	messagesJSON, err := json.Marshal(session.Messages)
	if err != nil {
		return err
	}

	stateJSON, err := json.Marshal(session.State)
	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE sessions SET messages = ?, state = ? WHERE id = ?",
		string(messagesJSON), string(stateJSON), session.ID)
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

// SetLogger sets the logger for a session (useful after loading from store)
func (session *Session) SetLogger(logger *slog.Logger) {
	session.logger = logger
}
