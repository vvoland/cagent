package servicecore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cagent/pkg/content"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolver_ResolveAgent(t *testing.T) {
	// Create a temporary directory for test agents
	tempDir, err := os.MkdirTemp("", "resolver-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	resolver, err := NewResolver(tempDir)
	require.NoError(t, err)

	t.Run("ResolveExistingFile", func(t *testing.T) {
		// Create a test agent file
		agentFile := filepath.Join(tempDir, "test-agent.yaml")
		err := os.WriteFile(agentFile, []byte("test agent content"), 0644)
		require.NoError(t, err)

		resolved, err := resolver.ResolveAgent(agentFile)
		require.NoError(t, err)
		assert.Equal(t, agentFile, resolved)
	})

	t.Run("ResolveRelativePath", func(t *testing.T) {
		// Create a test agent file in the agents directory
		agentFile := filepath.Join(tempDir, "relative-agent.yaml")
		err := os.WriteFile(agentFile, []byte("relative agent content"), 0644)
		require.NoError(t, err)

		resolved, err := resolver.ResolveAgent("relative-agent.yaml")
		require.NoError(t, err)
		assert.Equal(t, agentFile, resolved)
	})

	t.Run("ResolveNonExistentFile", func(t *testing.T) {
		// Should fail since we don't have a content store setup
		_, err := resolver.ResolveAgent("non-existent-agent.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found in files or store")
	})

	t.Run("RejectUnsafePath", func(t *testing.T) {
		// Create a file outside the temp directory
		outsideDir, err := os.MkdirTemp("", "outside-")
		require.NoError(t, err)
		defer os.RemoveAll(outsideDir)

		outsideFile := filepath.Join(outsideDir, "outside-agent.yaml")
		err = os.WriteFile(outsideFile, []byte("outside agent"), 0644)
		require.NoError(t, err)

		// Attempt to resolve should fail due to security restrictions
		_, err = resolver.ResolveAgent(outsideFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found in files or store")
	})

	t.Run("RejectPathTraversal", func(t *testing.T) {
		// Test path traversal attack
		_, err := resolver.ResolveAgent("../../../etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found in files or store")
	})
}

func TestResolver_ListFileAgents(t *testing.T) {
	// Create a temporary directory for test agents
	tempDir, err := os.MkdirTemp("", "resolver-list-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	resolver, err := NewResolver(tempDir)
	require.NoError(t, err)

	t.Run("EmptyDirectory", func(t *testing.T) {
		agents, err := resolver.ListFileAgents()
		require.NoError(t, err)
		assert.Len(t, agents, 0)
	})

	t.Run("WithAgentFiles", func(t *testing.T) {
		// Create test agent files
		agentFiles := []string{
			"echo-agent.yaml",
			"code-reviewer.yml",
			"sub/nested-agent.yaml",
		}

		for _, agentFile := range agentFiles {
			fullPath := filepath.Join(tempDir, agentFile)
			err := os.MkdirAll(filepath.Dir(fullPath), 0755)
			require.NoError(t, err)
			err = os.WriteFile(fullPath, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		// Create a non-YAML file that should be ignored
		err = os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("not an agent"), 0644)
		require.NoError(t, err)

		agents, err := resolver.ListFileAgents()
		require.NoError(t, err)
		assert.Len(t, agents, 3)

		// Check that all agents have correct properties
		for _, agent := range agents {
			assert.Equal(t, "file", agent.Source)
			assert.NotEmpty(t, agent.Name)
			assert.NotEmpty(t, agent.Path)
			assert.Contains(t, agent.Description, "File-based agent")
		}

		// Find specific agent
		var echoAgent *AgentInfo
		for _, agent := range agents {
			if agent.Name == "echo-agent" {
				echoAgent = &agent
				break
			}
		}
		require.NotNil(t, echoAgent)
		assert.Equal(t, "echo-agent", echoAgent.Name)
		assert.Contains(t, echoAgent.Path, "echo-agent.yaml")
	})

	t.Run("NonExistentDirectory", func(t *testing.T) {
		nonExistentResolver, err := NewResolver("/non/existent/path")
		require.NoError(t, err)

		agents, err := nonExistentResolver.ListFileAgents()
		require.NoError(t, err)
		assert.Len(t, agents, 0)
	})
}

func TestResolver_ListStoreAgents(t *testing.T) {
	// Create a temporary store for testing
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	require.NoError(t, err)

	resolver, err := NewResolverWithStore("/tmp", store)
	require.NoError(t, err)

	t.Run("EmptyStore", func(t *testing.T) {
		// Empty store should return empty list
		agents, err := resolver.ListStoreAgents()
		require.NoError(t, err)
		assert.Len(t, agents, 0)
	})
}

func TestResolver_PullAgent(t *testing.T) {

	// Create a temporary store for testing
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	require.NoError(t, err)

	resolver, err := NewResolverWithStore("/tmp", store)
	require.NoError(t, err)

	t.Run("InvalidReference", func(t *testing.T) {
		// Should fail with invalid registry reference
		err := resolver.PullAgent("invalid-reference")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pulling agent image")
	})
}

func TestResolver_FileExists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "resolver-fileexists-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	resolver, err := NewResolver(tempDir)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(tempDir, "test-file.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	assert.True(t, resolver.fileExists(testFile))
	assert.False(t, resolver.fileExists(filepath.Join(tempDir, "non-existent.txt")))
}

func TestResolver_IsPathSafe(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "resolver-security-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	resolver, err := NewResolver(tempDir)
	require.NoError(t, err)

	t.Run("SafePath", func(t *testing.T) {
		safePath := filepath.Join(tempDir, "safe-agent.yaml")
		err := resolver.isPathSafe(safePath)
		assert.NoError(t, err)
	})

	t.Run("PathTraversal", func(t *testing.T) {
		unsafePath := filepath.Join(tempDir, "../../../etc/passwd")
		err := resolver.isPathSafe(unsafePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path outside allowed root directory")
	})

	t.Run("OutsideRoot", func(t *testing.T) {
		outsidePath := "/etc/passwd"
		err := resolver.isPathSafe(outsidePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path outside allowed root directory")
	})
}

// Test that requires actual content store (integration test)
func TestResolver_FromStore_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary store for testing
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	require.NoError(t, err)

	resolver, err := NewResolverWithStore("/tmp", store)
	require.NoError(t, err)

	t.Run("NonExistentImage", func(t *testing.T) {
		_, err := resolver.fromStore("non-existent-image:latest")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "getting image from store")
	})
}
