// store.go implements multi-tenant session storage for cagent's servicecore architecture.
// This component provides persistent storage with client isolation for MCP and HTTP clients.
//
// Core Responsibilities:
// 1. Multi-Tenant Session Management:
//   - Client-scoped session storage with mandatory client_id isolation
//   - Backward-compatible database migration from single-tenant to multi-tenant
//   - Session lifecycle tracking (creation, retrieval, updates, deletion)
//   - Automatic cleanup when clients disconnect
//
// 2. Database Schema Evolution:
//   - Non-breaking migration adding client_id column with '__global' default
//   - Preserves existing sessions by assigning them to DEFAULT_CLIENT_ID
//   - Performance optimization through proper indexing on client_id
//   - PRAGMA-based schema introspection for safe migrations
//
// 3. Security and Isolation:
//   - All operations require explicit client_id to prevent cross-client access
//   - Input validation prevents empty client/session IDs
//   - Structured error handling with specific error types
//   - Defensive programming with proper resource cleanup
//
// 4. Integration with Existing Systems:
//   - Maintains compatibility with pkg/session data structures
//   - JSON serialization for message storage
//   - RFC3339 timestamp formatting for consistent time handling
//   - Bridge between servicecore types and legacy session storage
//
// Client ID Strategy:
// - MCP clients: Use real client ID from MCP session context
// - HTTP clients: Use '__global' constant until authentication is added
// - Legacy sessions: Automatically inherit '__global' client ID
//
// This component is essential for enabling true multi-tenant operation while
// maintaining backward compatibility with existing single-tenant deployments.
package servicecore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/docker/cagent/pkg/session"
	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrClientNotFound  = errors.New("client not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrEmptyClientID   = errors.New("client ID cannot be empty")
	ErrEmptySessionID  = errors.New("session ID cannot be empty")
)

// SQLiteStore implements Store using SQLite with multi-tenant support
type SQLiteStore struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewSQLiteStore creates a new SQLite store with multi-tenant support
func NewSQLiteStore(path string, logger *slog.Logger) (Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{
		db:     db,
		logger: logger,
	}

	if err := store.migrate(); err != nil {
		return nil, err
	}

	return store, nil
}

// migrate applies database schema migrations
func (s *SQLiteStore) migrate() error {
	ctx := context.Background()

	// Create sessions table if it doesn't exist (original schema)
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			messages TEXT,
			created_at TEXT,
			agent_spec TEXT
		)
	`)
	if err != nil {
		return err
	}

	// Check if client_id column exists
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info(sessions)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasClientID := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, dfltValue, pk interface{}
		err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
		if err != nil {
			return err
		}
		if name == "client_id" {
			hasClientID = true
			break
		}
	}

	// Add client_id column if it doesn't exist (non-breaking migration)
	if !hasClientID {
		s.logger.Info("Adding client_id column to sessions table")
		_, err = s.db.ExecContext(ctx, `
			ALTER TABLE sessions ADD COLUMN client_id TEXT DEFAULT '__global'
		`)
		if err != nil {
			return err
		}

		// Create index for client_id for performance
		_, err = s.db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_sessions_client_id ON sessions(client_id)
		`)
		if err != nil {
			return err
		}

		s.logger.Info("Successfully added client_id column with default '__global'")
	}

	return nil
}

// CreateClient creates a client record (no-op for SQLite, clients are implicit)
func (s *SQLiteStore) CreateClient(ctx context.Context, clientID string) error {
	if clientID == "" {
		return ErrEmptyClientID
	}
	// No explicit client table needed - clients are implicit through session records
	s.logger.Debug("Client created (implicit)", "client_id", clientID)
	return nil
}

// DeleteClient removes all sessions for a client
func (s *SQLiteStore) DeleteClient(ctx context.Context, clientID string) error {
	if clientID == "" {
		return ErrEmptyClientID
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE client_id = ?", clientID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	s.logger.Debug("Client sessions deleted", "client_id", clientID, "sessions_deleted", rowsAffected)
	return nil
}

// CreateSession creates a new agent session
func (s *SQLiteStore) CreateSession(ctx context.Context, clientID string, agentSession *AgentSession) error {
	if clientID == "" {
		return ErrEmptyClientID
	}
	if agentSession.ID == "" {
		return ErrEmptySessionID
	}

	// Convert AgentSession to session.Session format for storage
	sessionData := &session.Session{
		ID:        agentSession.ID,
		Messages:  []session.Message{}, // Start with empty messages
		CreatedAt: agentSession.Created,
	}

	messagesJSON, err := json.Marshal(sessionData.Messages)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO sessions (id, client_id, messages, created_at, agent_spec) VALUES (?, ?, ?, ?, ?)",
		agentSession.ID, clientID, string(messagesJSON), agentSession.Created.Format(time.RFC3339), agentSession.AgentSpec)

	if err != nil {
		return err
	}

	s.logger.Debug("Session created", "client_id", clientID, "session_id", agentSession.ID)
	return nil
}

// GetSession retrieves a session by client and session ID
func (s *SQLiteStore) GetSession(ctx context.Context, clientID, sessionID string) (*AgentSession, error) {
	if clientID == "" {
		return nil, ErrEmptyClientID
	}
	if sessionID == "" {
		return nil, ErrEmptySessionID
	}

	row := s.db.QueryRowContext(ctx,
		"SELECT id, messages, created_at, agent_spec FROM sessions WHERE id = ? AND client_id = ?",
		sessionID, clientID)

	var messagesJSON, createdAtStr, agentSpec string
	var id string

	err := row.Scan(&id, &messagesJSON, &createdAtStr, &agentSpec)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	// Parse messages (though we don't store them in AgentSession directly)
	var messages []session.Message
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return nil, err
	}

	agentSession := &AgentSession{
		ID:        id,
		ClientID:  clientID,
		AgentSpec: agentSpec,
		Created:   createdAt,
		LastUsed:  createdAt, // Will be updated by manager
		// Runtime and Session will be set by manager
	}

	return agentSession, nil
}

// ListSessions lists all sessions for a client
func (s *SQLiteStore) ListSessions(ctx context.Context, clientID string) ([]*AgentSession, error) {
	if clientID == "" {
		return nil, ErrEmptyClientID
	}

	rows, err := s.db.QueryContext(ctx,
		"SELECT id, messages, created_at, agent_spec FROM sessions WHERE client_id = ? ORDER BY created_at DESC",
		clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*AgentSession, 0)
	for rows.Next() {
		var messagesJSON, createdAtStr, agentSpec string
		var id string

		err := rows.Scan(&id, &messagesJSON, &createdAtStr, &agentSpec)
		if err != nil {
			return nil, err
		}

		createdAt, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, err
		}

		agentSession := &AgentSession{
			ID:        id,
			ClientID:  clientID,
			AgentSpec: agentSpec,
			Created:   createdAt,
			LastUsed:  createdAt, // Will be updated by manager
		}

		sessions = append(sessions, agentSession)
	}

	return sessions, nil
}

// UpdateSession updates an existing session
func (s *SQLiteStore) UpdateSession(ctx context.Context, clientID string, agentSession *AgentSession) error {
	if clientID == "" {
		return ErrEmptyClientID
	}
	if agentSession.ID == "" {
		return ErrEmptySessionID
	}

	// For now, we mainly track the session existence
	// The actual message updates are handled by the session.Session in memory
	result, err := s.db.ExecContext(ctx,
		"UPDATE sessions SET created_at = created_at WHERE id = ? AND client_id = ?",
		agentSession.ID, clientID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteSession deletes a session
func (s *SQLiteStore) DeleteSession(ctx context.Context, clientID, sessionID string) error {
	if clientID == "" {
		return ErrEmptyClientID
	}
	if sessionID == "" {
		return ErrEmptySessionID
	}

	result, err := s.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE id = ? AND client_id = ?",
		sessionID, clientID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrSessionNotFound
	}

	s.logger.Debug("Session deleted", "client_id", clientID, "session_id", sessionID)
	return nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
