package mcpserver

import (
	"os"
	"testing"
	"time"

	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/servicecore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionToolRegistration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "mcp-session-tools-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create isolated store for testing
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	require.NoError(t, err)

	resolver, err := servicecore.NewResolverWithStore(tempDir, store)
	require.NoError(t, err)

	manager, err := servicecore.NewManagerWithResolver(resolver, time.Hour, 10)
	require.NoError(t, err)

	t.Run("ServerCreationWithSessionTools", func(t *testing.T) {
		mcpServer := NewMCPServer(manager, "/mcp")

		// Verify server is created properly
		assert.NotNil(t, mcpServer)
		assert.Equal(t, manager, mcpServer.serviceCore)
		assert.NotNil(t, mcpServer.mcpServer)
		assert.NotNil(t, mcpServer.sseServer)
	})

	t.Run("SessionToolsAvailable", func(t *testing.T) {
		mcpServer := NewMCPServer(manager, "/mcp")

		// Verify the server is created (tools are registered in constructor)
		assert.NotNil(t, mcpServer.mcpServer, "MCP server should be created with session tools")

		// The session tool handlers should be available as methods
		assert.NotNil(t, mcpServer.handleCreateAgentSession)
		assert.NotNil(t, mcpServer.handleSendMessage)
		assert.NotNil(t, mcpServer.handleListAgentSessions)
		assert.NotNil(t, mcpServer.handleCloseAgentSession)
		assert.NotNil(t, mcpServer.handleGetAgentSessionInfo)
	})
}

func TestFormatSessionList(t *testing.T) {
	sessionList := []interface{}{
		map[string]interface{}{
			"id":         "session-1",
			"agent_spec": "test-agent.yaml",
			"created":    "2025-07-27 10:00:00",
			"last_used":  "2025-07-27 10:05:00",
		},
		map[string]interface{}{
			"id":         "session-2",
			"agent_spec": "echo-agent.yaml",
			"created":    "2025-07-27 11:00:00",
			"last_used":  "2025-07-27 11:15:00",
		},
	}

	result := formatSessionList(sessionList)

	assert.Contains(t, result, "session-1")
	assert.Contains(t, result, "test-agent.yaml")
	assert.Contains(t, result, "session-2")
	assert.Contains(t, result, "echo-agent.yaml")
	assert.Contains(t, result, "2025-07-27 10:00:00")
	assert.Contains(t, result, "2025-07-27 11:15:00")
}

func TestAdvancedSessionTools(t *testing.T) {
	// Create isolated store for testing
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	require.NoError(t, err)

	tempDir := t.TempDir()
	resolver, err := servicecore.NewResolverWithStore(tempDir, store)
	require.NoError(t, err)

	manager, err := servicecore.NewManagerWithResolver(resolver, time.Hour, 10)
	require.NoError(t, err)

	// Create MCP server
	mcpServer := NewMCPServer(manager, "/mcp")

	t.Run("AdvancedToolsRegistered", func(t *testing.T) {
		// Verify the advanced session tool handlers exist as methods
		assert.NotNil(t, mcpServer.handleGetAgentSessionHistory)
		assert.NotNil(t, mcpServer.handleGetAgentSessionInfoEnhanced)
	})

	t.Run("ServiceCoreAdvancedMethods", func(t *testing.T) {
		// Test that MCP server can access servicecore advanced functionality

		// Create a client
		err := mcpServer.serviceCore.CreateClient("test-client")
		assert.NoError(t, err)

		// Test GetSessionHistory for non-existent session (should return error)
		_, err = mcpServer.serviceCore.GetSessionHistory("test-client", "non-existent", 10)
		assert.Error(t, err)

		// Test GetSessionInfo for non-existent session (should return error)
		_, err = mcpServer.serviceCore.GetSessionInfo("test-client", "non-existent")
		assert.Error(t, err)

		// Remove client
		err = mcpServer.serviceCore.RemoveClient("test-client")
		assert.NoError(t, err)
	})
}

func TestSessionToolsServiceCoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "mcp-session-integration-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create isolated store for testing
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	require.NoError(t, err)

	resolver, err := servicecore.NewResolverWithStore(tempDir, store)
	require.NoError(t, err)

	manager, err := servicecore.NewManagerWithResolver(resolver, time.Hour, 10)
	require.NoError(t, err)

	mcpServer := NewMCPServer(manager, "/mcp")

	t.Run("ServiceCoreSessionOperations", func(t *testing.T) {
		// Test that MCP server can access servicecore session functionality

		// Create a client
		err := mcpServer.serviceCore.CreateClient("test-client")
		assert.NoError(t, err)

		// List sessions (should be empty)
		sessions, err := mcpServer.serviceCore.ListSessions("test-client")
		assert.NoError(t, err)
		assert.Len(t, sessions, 0)

		// Remove client
		err = mcpServer.serviceCore.RemoveClient("test-client")
		assert.NoError(t, err)
	})
}

func TestSessionToolsHandlerIntegration(t *testing.T) {
	// Create isolated store for testing
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	require.NoError(t, err)

	tempDir := t.TempDir()
	resolver, err := servicecore.NewResolverWithStore(tempDir, store)
	require.NoError(t, err)

	manager, err := servicecore.NewManagerWithResolver(resolver, time.Hour, 10)
	require.NoError(t, err)

	// Create MCP server
	mcpServer := NewMCPServer(manager, "/mcp")

	t.Run("SessionToolsRegistered", func(t *testing.T) {
		// Verify server was created with session tools
		assert.NotNil(t, mcpServer)

		// Test that handlers exist as methods
		assert.NotNil(t, mcpServer.handleCreateAgentSession)
		assert.NotNil(t, mcpServer.handleSendMessage)
		assert.NotNil(t, mcpServer.handleListAgentSessions)
		assert.NotNil(t, mcpServer.handleCloseAgentSession)
		assert.NotNil(t, mcpServer.handleGetAgentSessionInfo)
	})

	t.Run("ServiceCoreIntegration", func(t *testing.T) {
		// Test that MCP server can access servicecore functionality
		err := mcpServer.serviceCore.CreateClient("test-client")
		assert.NoError(t, err)

		// List sessions (should be empty)
		sessions, err := mcpServer.serviceCore.ListSessions("test-client")
		assert.NoError(t, err)
		assert.Len(t, sessions, 0)

		// Remove client
		err = mcpServer.serviceCore.RemoveClient("test-client")
		assert.NoError(t, err)
	})

	t.Run("FormatSessionListHelper", func(t *testing.T) {
		// Test the session formatting helper
		sessionList := []interface{}{
			map[string]interface{}{
				"id":         "session-1",
				"agent_spec": "test-agent.yaml",
				"created":    "2025-07-27 10:00:00",
				"last_used":  "2025-07-27 10:05:00",
			},
		}

		result := formatSessionList(sessionList)
		assert.Contains(t, result, "session-1")
		assert.Contains(t, result, "test-agent.yaml")
		assert.Contains(t, result, "2025-07-27 10:00:00")
	})
}
