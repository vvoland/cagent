// server_test.go provides unit tests for the MCP server implementation
// This file tests the server creation, tool registration, and basic functionality
// without requiring a full MCP client integration.
//
package mcpserver

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/docker/cagent/pkg/servicecore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	
	// Create a mock servicecore manager
	tempDir, err := os.MkdirTemp("", "mcpserver-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	serviceCore, err := servicecore.NewManager(tempDir, time.Hour, 100, logger)
	require.NoError(t, err)

	t.Run("ServerCreation", func(t *testing.T) {
		mcpServer := NewMCPServer(serviceCore, logger, "/mcp")
		
		assert.NotNil(t, mcpServer)
		assert.Equal(t, serviceCore, mcpServer.serviceCore)
		assert.Equal(t, logger, mcpServer.logger)
		assert.NotNil(t, mcpServer.mcpServer)
		assert.NotNil(t, mcpServer.sseServer)
	})

	t.Run("ServerConfiguration", func(t *testing.T) {
		mcpServer := NewMCPServer(serviceCore, logger, "/mcp")
		
		// Verify server is properly configured
		assert.NotNil(t, mcpServer.mcpServer, "MCP server should be created")
		assert.NotNil(t, mcpServer.sseServer, "SSE server should be created")
	})
}

func TestMCPServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	
	// Create a temporary directory for test agents
	tempDir, err := os.MkdirTemp("", "mcpserver-integration-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create a test agent file
	testAgent := `
agents:
  root:
    name: test-agent
    description: Test agent for MCP integration
    model: test-model
    instruction: You are a test agent

models:
  test-model:
    provider: openai
    type: gpt-4
`
	agentFile := tempDir + "/test-agent.yaml"
	err = os.WriteFile(agentFile, []byte(testAgent), 0644)
	require.NoError(t, err)
	
	serviceCore, err := servicecore.NewManager(tempDir, time.Hour, 100, logger)
	require.NoError(t, err)

	t.Run("ServiceCoreIntegration", func(t *testing.T) {
		mcpServer := NewMCPServer(serviceCore, logger, "/mcp")
		
		// Test that the server can access servicecore functionality
		agents, err := mcpServer.serviceCore.ListAgents("files")
		assert.NoError(t, err)
		assert.Len(t, agents, 1)
		assert.Equal(t, "test-agent", agents[0].Name)
	})
}

// Note: Full MCP client integration tests would require setting up
// an actual MCP client and testing the SSE endpoints. This would be
// better suited for end-to-end testing rather than unit tests.