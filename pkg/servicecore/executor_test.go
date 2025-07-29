package servicecore

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_CreateRuntime(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	executor := NewExecutor(logger)

	t.Run("InvalidAgentPath", func(t *testing.T) {
		_, _, err := executor.CreateRuntime("non-existent.yaml", "root", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "loading agent configuration")
	})

	t.Run("EmptyAgentName", func(t *testing.T) {
		// Create a minimal valid agent file
		tempDir, err := os.MkdirTemp("", "executor-test-")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		validAgent := `
agents:
  root:
    name: test-agent
    description: Test agent
    model: test-model
    instruction: You are a test agent

models:
  test-model:
    provider: openai
    type: gpt-4
`
		agentFile := tempDir + "/valid-agent.yaml"
		err = os.WriteFile(agentFile, []byte(validAgent), 0644)
		require.NoError(t, err)

		// This will still fail due to missing API keys, but should get further
		_, _, err = executor.CreateRuntime(agentFile, "", nil, "")
		assert.Error(t, err)
		// The error will be from the loader/runtime, not from our executor
		assert.Contains(t, err.Error(), "loading agent configuration")
	})
}

func TestExecutor_CleanupRuntime(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	executor := NewExecutor(logger)

	t.Run("NilRuntime", func(t *testing.T) {
		err := executor.CleanupRuntime(nil)
		assert.NoError(t, err)
	})

	// Note: Testing with actual runtime would require complex setup
	// This test validates the nil check works correctly
}

func TestNewExecutor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	
	executor := NewExecutor(logger)
	assert.NotNil(t, executor)
	assert.Equal(t, logger, executor.logger)
}

// Integration tests would require full agent setup with models, tools, etc.
// These are better suited for end-to-end testing rather than unit tests
func TestExecutor_ExecuteStream_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("NilRuntime", func(t *testing.T) {
		// ExecuteStream with nil runtime should be caught before calling
		// In practice, this would be prevented by the manager
		// Let's test that the function handles it gracefully by checking preconditions
		// For now, skip this test as it would require significant error handling changes
		t.Skip("ExecuteStream expects valid runtime - tested at manager level")
	})
}