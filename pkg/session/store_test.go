package session

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/chat"
)

func TestStoreAgentName(t *testing.T) {
	// Create a temporary database file
	tempDB := "test_store.db"
	defer os.Remove(tempDB)

	// Create the store
	store, err := NewSQLiteSessionStore(tempDB)
	require.NoError(t, err)
	defer store.(*SQLiteSessionStore).Close()

	// Create test agents
	testAgent1 := agent.New("test-agent-1", "test prompt 1")
	testAgent2 := agent.New("test-agent-2", "test prompt 2")

	// Create a session with messages from different agents
	session := &Session{
		ID: "test-session",
		Messages: []Message{
			UserMessage("demo-agent", "Hello"),
			NewAgentMessage(testAgent1, &chat.Message{
				Role:    chat.MessageRoleAssistant,
				Content: "Hello from test-agent-1",
			}),
			NewAgentMessage(testAgent2, &chat.Message{
				Role:    chat.MessageRoleUser,
				Content: "Another message from test-agent-2",
			}),
		},
		CreatedAt: time.Now(),
	}

	// Store the session
	err = store.AddSession(context.Background(), session)
	require.NoError(t, err)

	// Retrieve the session
	retrievedSession, err := store.GetSession(context.Background(), "test-session")
	require.NoError(t, err)
	require.NotNil(t, retrievedSession)

	// Verify the agent names are correctly stored and retrieved
	assert.Equal(t, 3, len(retrievedSession.Messages))

	// First message should be user message with empty agent name
	assert.Equal(t, "demo-agent", retrievedSession.Messages[0].AgentFilename)
	assert.Equal(t, "", retrievedSession.Messages[0].AgentName)
	assert.Equal(t, "Hello", retrievedSession.Messages[0].Message.Content)

	// Second message should have the first agent's name
	assert.Equal(t, "", retrievedSession.Messages[1].AgentFilename)
	assert.Equal(t, "test-agent-1", retrievedSession.Messages[1].AgentName)
	assert.Equal(t, "Hello from test-agent-1", retrievedSession.Messages[1].Message.Content)

	// Third message should have the second agent's name
	assert.Equal(t, "", retrievedSession.Messages[2].AgentFilename)
	assert.Equal(t, "test-agent-2", retrievedSession.Messages[2].AgentName)
	assert.Equal(t, "Another message from test-agent-2", retrievedSession.Messages[2].Message.Content)
}

func TestStoreMultipleAgents(t *testing.T) {
	// Create a temporary database file
	tempDB := "test_store_multi.db"
	defer os.Remove(tempDB)

	// Create the store
	store, err := NewSQLiteSessionStore(tempDB)
	require.NoError(t, err)
	defer store.(*SQLiteSessionStore).Close()

	// Create multiple test agents
	agent1 := agent.New("agent-1", "agent 1 prompt")
	agent2 := agent.New("agent-2", "agent 2 prompt")

	// Create a session with messages from different agents
	session := &Session{
		ID:        "multi-agent-session",
		CreatedAt: time.Now(),
		Messages: []Message{
			UserMessage("demo", "Start conversation"),
			NewAgentMessage(agent1, &chat.Message{
				Role:    chat.MessageRoleAssistant,
				Content: "Response from agent 1",
			}),
			NewAgentMessage(agent2, &chat.Message{
				Role:    chat.MessageRoleAssistant,
				Content: "Response from agent 2",
			}),
		},
	}

	// Store the session
	err = store.AddSession(context.Background(), session)
	require.NoError(t, err)

	// Retrieve the session
	retrievedSession, err := store.GetSession(context.Background(), "multi-agent-session")
	require.NoError(t, err)
	require.NotNil(t, retrievedSession)

	// Verify the agent names are correctly stored and retrieved
	assert.Equal(t, 3, len(retrievedSession.Messages))

	// First message should be user message with empty agent name
	assert.Equal(t, "demo", retrievedSession.Messages[0].AgentFilename)
	assert.Equal(t, "", retrievedSession.Messages[0].AgentName)

	// Second message should have agent-1 name
	assert.Equal(t, "", retrievedSession.Messages[1].AgentFilename)
	assert.Equal(t, "agent-1", retrievedSession.Messages[1].AgentName)
	assert.Equal(t, "Response from agent 1", retrievedSession.Messages[1].Message.Content)

	// Third message should have agent-2 name
	assert.Equal(t, "", retrievedSession.Messages[2].AgentFilename)
	assert.Equal(t, "agent-2", retrievedSession.Messages[2].AgentName)
	assert.Equal(t, "Response from agent 2", retrievedSession.Messages[2].Message.Content)
}

func TestGetSessions(t *testing.T) {
	// Create a temporary database file
	tempDB := "test_get_sessions.db"
	defer os.Remove(tempDB)

	// Create the store
	store, err := NewSQLiteSessionStore(tempDB)
	require.NoError(t, err)
	defer store.(*SQLiteSessionStore).Close()

	// Create a test agent
	testAgent := agent.New("test-agent", "test prompt")

	// Create multiple sessions
	session1 := &Session{
		ID: "session-1",
		Messages: []Message{
			NewAgentMessage(testAgent, &chat.Message{
				Role:    chat.MessageRoleAssistant,
				Content: "Message from session 1",
			}),
		},
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	session2 := &Session{
		ID: "session-2",
		Messages: []Message{
			NewAgentMessage(testAgent, &chat.Message{
				Role:    chat.MessageRoleAssistant,
				Content: "Message from session 2",
			}),
		},
		CreatedAt: time.Now(),
	}

	// Store the sessions
	err = store.AddSession(context.Background(), session1)
	require.NoError(t, err)
	err = store.AddSession(context.Background(), session2)
	require.NoError(t, err)

	// Retrieve all sessions
	sessions, err := store.GetSessions(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, len(sessions))

	// Verify agent names are preserved in all sessions
	for _, session := range sessions {
		assert.Equal(t, 1, len(session.Messages))
		assert.Equal(t, "test-agent", session.Messages[0].AgentName)
	}
}

func TestStoreAgentNameJSON(t *testing.T) {
	// Create a temporary database file
	tempDB := "test_store_json.db"
	defer os.Remove(tempDB)

	// Create the store
	store, err := NewSQLiteSessionStore(tempDB)
	require.NoError(t, err)
	defer store.(*SQLiteSessionStore).Close()

	// Create test agents
	agent1 := agent.New("my-agent", "test prompt")
	agent2 := agent.New("another-agent", "another prompt")

	// Create a session with messages from different agents
	session := &Session{
		ID: "json-test-session",
		Messages: []Message{
			UserMessage("demo-agent", "User input"),
			NewAgentMessage(agent1, &chat.Message{
				Role:    chat.MessageRoleAssistant,
				Content: "Response from my-agent",
			}),
			NewAgentMessage(agent2, &chat.Message{
				Role:    chat.MessageRoleAssistant,
				Content: "Response from another-agent",
			}),
		},
		CreatedAt: time.Now(),
	}

	// Store the session
	err = store.AddSession(context.Background(), session)
	require.NoError(t, err)

	// Retrieve the session
	retrievedSession, err := store.GetSession(context.Background(), "json-test-session")
	require.NoError(t, err)
	require.NotNil(t, retrievedSession)

	// Verify specific agent filenames are correctly stored and retrieved
	assert.Equal(t, "demo-agent", retrievedSession.Messages[0].AgentFilename) // User message
	assert.Equal(t, "", retrievedSession.Messages[1].AgentFilename)           // First agent
	assert.Equal(t, "", retrievedSession.Messages[2].AgentFilename)           // Second agent

	// Verify specific agent names are correctly stored and retrieved
	assert.Equal(t, "", retrievedSession.Messages[0].AgentName)              // User message
	assert.Equal(t, "my-agent", retrievedSession.Messages[1].AgentName)      // First agent
	assert.Equal(t, "another-agent", retrievedSession.Messages[2].AgentName) // Second agent
}
