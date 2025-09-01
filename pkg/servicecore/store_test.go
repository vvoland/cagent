package servicecore

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStore_Migration(t *testing.T) {
	// Create a temporary database file
	tempDB := "test_store_migration.db"
	defer os.Remove(tempDB)

	// Create the store - should apply migration
	store, err := NewSQLiteStore(tempDB)
	require.NoError(t, err)
	defer store.(*SQLiteStore).Close()

	// Verify migration was applied by creating a client
	ctx := t.Context()
	err = store.CreateClient(ctx, "test-client")
	assert.NoError(t, err)
}

func TestSQLiteStore_ClientOperations(t *testing.T) {
	tempDB := "test_store_client.db"
	defer os.Remove(tempDB)

	store, err := NewSQLiteStore(tempDB)
	require.NoError(t, err)
	defer store.(*SQLiteStore).Close()

	ctx := context.Background()

	t.Run("CreateClient", func(t *testing.T) {
		err := store.CreateClient(ctx, "client-1")
		assert.NoError(t, err)

		// Empty client ID should fail
		err = store.CreateClient(ctx, "")
		assert.Equal(t, ErrEmptyClientID, err)
	})

	t.Run("DeleteClient", func(t *testing.T) {
		// First create a client with sessions
		err := store.CreateClient(ctx, "client-to-delete")
		require.NoError(t, err)

		session := &AgentSession{
			ID:        "session-1",
			ClientID:  "client-to-delete",
			AgentSpec: "test-agent",
			Created:   time.Now(),
		}
		err = store.CreateSession(ctx, "client-to-delete", session)
		require.NoError(t, err)

		// Delete client should remove all sessions
		err = store.DeleteClient(ctx, "client-to-delete")
		assert.NoError(t, err)

		// Verify session was deleted
		_, err = store.GetSession(ctx, "client-to-delete", "session-1")
		assert.Equal(t, ErrSessionNotFound, err)

		// Empty client ID should fail
		err = store.DeleteClient(ctx, "")
		assert.Equal(t, ErrEmptyClientID, err)
	})
}

func TestSQLiteStore_SessionOperations(t *testing.T) {
	tempDB := "test_store_session.db"
	defer os.Remove(tempDB)

	store, err := NewSQLiteStore(tempDB)
	require.NoError(t, err)
	defer store.(*SQLiteStore).Close()

	ctx := context.Background()

	// Create a client first
	err = store.CreateClient(ctx, "test-client")
	require.NoError(t, err)

	session := &AgentSession{
		ID:        "test-session",
		ClientID:  "test-client",
		AgentSpec: "test-agent.yaml",
		Created:   time.Now(),
		LastUsed:  time.Now(),
	}

	t.Run("CreateSession", func(t *testing.T) {
		err := store.CreateSession(ctx, "test-client", session)
		assert.NoError(t, err)

		// Empty client ID should fail
		err = store.CreateSession(ctx, "", session)
		assert.Equal(t, ErrEmptyClientID, err)

		// Empty session ID should fail
		emptySession := &AgentSession{ID: "", ClientID: "test-client"}
		err = store.CreateSession(ctx, "test-client", emptySession)
		assert.Equal(t, ErrEmptySessionID, err)
	})

	t.Run("GetSession", func(t *testing.T) {
		retrieved, err := store.GetSession(ctx, "test-client", "test-session")
		require.NoError(t, err)

		assert.Equal(t, session.ID, retrieved.ID)
		assert.Equal(t, session.ClientID, retrieved.ClientID)
		assert.Equal(t, session.AgentSpec, retrieved.AgentSpec)

		// Non-existent session should fail
		_, err = store.GetSession(ctx, "test-client", "non-existent")
		assert.Equal(t, ErrSessionNotFound, err)

		// Wrong client should fail
		_, err = store.GetSession(ctx, "wrong-client", "test-session")
		assert.Equal(t, ErrSessionNotFound, err)

		// Empty client ID should fail
		_, err = store.GetSession(ctx, "", "test-session")
		assert.Equal(t, ErrEmptyClientID, err)

		// Empty session ID should fail
		_, err = store.GetSession(ctx, "test-client", "")
		assert.Equal(t, ErrEmptySessionID, err)
	})

	t.Run("ListSessions", func(t *testing.T) {
		// Create another session with a later timestamp
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
		session2 := &AgentSession{
			ID:        "test-session-2",
			ClientID:  "test-client",
			AgentSpec: "another-agent.yaml",
			Created:   time.Now(),
		}
		err := store.CreateSession(ctx, "test-client", session2)
		require.NoError(t, err)

		sessions, err := store.ListSessions(ctx, "test-client")
		require.NoError(t, err)
		assert.Len(t, sessions, 2)

		// Should be ordered by created_at DESC (most recent first)
		// Find which session is which
		var sess1, sess2 *AgentSession
		for _, s := range sessions {
			if s.ID == "test-session" {
				sess1 = s
			} else if s.ID == "test-session-2" {
				sess2 = s
			}
		}
		require.NotNil(t, sess1)
		require.NotNil(t, sess2)

		// session2 should be listed first (more recent)
		assert.True(t, sessions[0].Created.After(sessions[1].Created) || sessions[0].Created.Equal(sessions[1].Created))

		// Empty client should return empty list
		sessions, err = store.ListSessions(ctx, "non-existent-client")
		require.NoError(t, err)
		assert.Len(t, sessions, 0)

		// Empty client ID should fail
		_, err = store.ListSessions(ctx, "")
		assert.Equal(t, ErrEmptyClientID, err)
	})

	t.Run("UpdateSession", func(t *testing.T) {
		err := store.UpdateSession(ctx, "test-client", session)
		assert.NoError(t, err)

		// Non-existent session should fail
		nonExistent := &AgentSession{ID: "non-existent", ClientID: "test-client"}
		err = store.UpdateSession(ctx, "test-client", nonExistent)
		assert.Equal(t, ErrSessionNotFound, err)

		// Empty client ID should fail
		err = store.UpdateSession(ctx, "", session)
		assert.Equal(t, ErrEmptyClientID, err)

		// Empty session ID should fail
		emptySession := &AgentSession{ID: "", ClientID: "test-client"}
		err = store.UpdateSession(ctx, "test-client", emptySession)
		assert.Equal(t, ErrEmptySessionID, err)
	})

	t.Run("DeleteSession", func(t *testing.T) {
		err := store.DeleteSession(ctx, "test-client", "test-session-2")
		assert.NoError(t, err)

		// Verify deletion
		_, err = store.GetSession(ctx, "test-client", "test-session-2")
		assert.Equal(t, ErrSessionNotFound, err)

		// Non-existent session should fail
		err = store.DeleteSession(ctx, "test-client", "non-existent")
		assert.Equal(t, ErrSessionNotFound, err)

		// Empty client ID should fail
		err = store.DeleteSession(ctx, "", "test-session")
		assert.Equal(t, ErrEmptyClientID, err)

		// Empty session ID should fail
		err = store.DeleteSession(ctx, "test-client", "")
		assert.Equal(t, ErrEmptySessionID, err)
	})
}

func TestSQLiteStore_ClientIsolation(t *testing.T) {
	tempDB := "test_store_isolation.db"
	defer os.Remove(tempDB)

	store, err := NewSQLiteStore(tempDB)
	require.NoError(t, err)
	defer store.(*SQLiteStore).Close()

	ctx := context.Background()

	// Create two clients
	err = store.CreateClient(ctx, "client-a")
	require.NoError(t, err)
	err = store.CreateClient(ctx, "client-b")
	require.NoError(t, err)

	// Create sessions for each client with different session IDs
	sessionA := &AgentSession{
		ID:        "session-a",
		ClientID:  "client-a",
		AgentSpec: "agent-a.yaml",
		Created:   time.Now(),
	}
	sessionB := &AgentSession{
		ID:        "session-b",
		ClientID:  "client-b",
		AgentSpec: "agent-b.yaml",
		Created:   time.Now(),
	}

	err = store.CreateSession(ctx, "client-a", sessionA)
	require.NoError(t, err)
	err = store.CreateSession(ctx, "client-b", sessionB)
	require.NoError(t, err)

	// Client A should only see its own session
	retrieved, err := store.GetSession(ctx, "client-a", "session-a")
	require.NoError(t, err)
	assert.Equal(t, "agent-a.yaml", retrieved.AgentSpec)

	// Client B should only see its own session
	retrieved, err = store.GetSession(ctx, "client-b", "session-b")
	require.NoError(t, err)
	assert.Equal(t, "agent-b.yaml", retrieved.AgentSpec)

	// Client A cannot access Client B's session
	_, err = store.GetSession(ctx, "client-a", "session-b")
	assert.Equal(t, ErrSessionNotFound, err)

	// List sessions should be client-scoped
	sessionsA, err := store.ListSessions(ctx, "client-a")
	require.NoError(t, err)
	assert.Len(t, sessionsA, 1)
	assert.Equal(t, "agent-a.yaml", sessionsA[0].AgentSpec)

	sessionsB, err := store.ListSessions(ctx, "client-b")
	require.NoError(t, err)
	assert.Len(t, sessionsB, 1)
	assert.Equal(t, "agent-b.yaml", sessionsB[0].AgentSpec)
}
