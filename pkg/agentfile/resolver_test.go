package agentfile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/aliases"
	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/oci"
)

func TestOciRefToFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ociRef   string
		expected string
	}{
		{
			name:     "simple reference",
			ociRef:   "myagent",
			expected: "myagent.yaml",
		},
		{
			name:     "reference with registry and tag",
			ociRef:   "docker.io/myorg/agent:v1",
			expected: "docker.io_myorg_agent_v1.yaml",
		},
		{
			name:     "localhost with port",
			ociRef:   "localhost:5000/test",
			expected: "localhost_5000_test.yaml",
		},
		{
			name:     "reference with digest",
			ociRef:   "myregistry.io/org/app@sha256:abc123",
			expected: "myregistry.io_org_app_sha256_abc123.yaml",
		},
		{
			name:     "already has .yaml extension",
			ociRef:   "myagent.yaml",
			expected: "myagent.yaml",
		},
		{
			name:     "complex path",
			ociRef:   "registry.example.com:443/project/subproject/agent:latest",
			expected: "registry.example.com_443_project_subproject_agent_latest.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := OciRefToFilename(tt.ociRef)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveAgentFile_LocalFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test-agent.yaml")
	yamlContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Test agent
    instruction: You are a test agent
`
	require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0o644))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	resolved, err := Resolve(ctx, nil, yamlFile)
	require.NoError(t, err)

	absPath, err := filepath.Abs(yamlFile)
	require.NoError(t, err)
	assert.Equal(t, absPath, resolved)
}

func TestResolveAgentFile_OCIRef_ConsistentFilename(t *testing.T) {
	storeDir := t.TempDir()
	t.Setenv("CAGENT_CONTENT_STORE", storeDir)

	store, err := content.NewStore()
	require.NoError(t, err)

	agentContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Test OCI agent
    instruction: You are a test OCI agent
`
	agentFile := filepath.Join(t.TempDir(), "oci-agent.yaml")
	require.NoError(t, os.WriteFile(agentFile, []byte(agentContent), 0o644))

	// Package as OCI artifact
	ociRef := "test.registry.io/myorg/testagent:v1"
	_, err = oci.PackageFileAsOCIToStore(t.Context(), agentFile, ociRef, store)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// First resolution
	resolved1, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)
	assert.NotEmpty(t, resolved1)

	content1, err := os.ReadFile(resolved1)
	require.NoError(t, err)
	assert.Equal(t, `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Test OCI agent
    instruction: You are a test OCI agent
`, string(content1))

	// Expected filename based on OCI ref
	expectedFilename := OciRefToFilename(ociRef)
	assert.Equal(t, expectedFilename, filepath.Base(resolved1))

	// Store the first resolved path
	firstResolvedPath := resolved1

	// Second resolution (simulating a reload)
	resolved2, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)

	// Should return the SAME filename
	assert.Equal(t, resolved1, resolved2, "Subsequent resolutions should return the same file path")
	assert.Equal(t, filepath.Base(resolved1), filepath.Base(resolved2), "Filenames should be identical")

	content2, err := os.ReadFile(resolved2)
	require.NoError(t, err)
	assert.Equal(t, agentContent, string(content2))

	// Update the agent content in the OCI store
	updatedContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o-mini
    description: Updated test OCI agent
    instruction: You are an updated test OCI agent
`
	updatedFile := filepath.Join(t.TempDir(), "updated-agent.yaml")
	require.NoError(t, os.WriteFile(updatedFile, []byte(updatedContent), 0o644))
	_, err = oci.PackageFileAsOCIToStore(t.Context(), updatedFile, ociRef, store)
	require.NoError(t, err)

	// Third resolution (simulating reload after update)
	resolved3, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)

	// Should STILL use the same filename
	assert.Equal(t, firstResolvedPath, resolved3, "Even after OCI update, should use same file path")

	// But content should be updated
	content3, err := os.ReadFile(resolved3)
	require.NoError(t, err)
	assert.Equal(t, updatedContent, string(content3), "Content should be updated from OCI store")
}

func TestResolveAgentFile_MultipleOCIRefs_DifferentFilenames(t *testing.T) {
	storeDir := t.TempDir()
	t.Setenv("CAGENT_CONTENT_STORE", storeDir)

	store, err := content.NewStore()
	require.NoError(t, err)

	agent1Content := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Agent 1
    instruction: You are agent 1
`
	agent2Content := `version: "1"
agents:
  root:
    model: anthropic/claude-sonnet-4-0
    description: Agent 2
    instruction: You are agent 2
`

	agent1File := filepath.Join(t.TempDir(), "agent1.yaml")
	agent2File := filepath.Join(t.TempDir(), "agent2.yaml")
	require.NoError(t, os.WriteFile(agent1File, []byte(agent1Content), 0o644))
	require.NoError(t, os.WriteFile(agent2File, []byte(agent2Content), 0o644))

	// Package as different OCI artifacts
	ociRef1 := "test.io/org/agent1:v1"
	ociRef2 := "test.io/org/agent2:v1"
	_, err = oci.PackageFileAsOCIToStore(t.Context(), agent1File, ociRef1, store)
	require.NoError(t, err)
	_, err = oci.PackageFileAsOCIToStore(t.Context(), agent2File, ociRef2, store)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Resolve both OCI refs
	resolved1, err := Resolve(ctx, nil, ociRef1)
	require.NoError(t, err)

	resolved2, err := Resolve(ctx, nil, ociRef2)
	require.NoError(t, err)

	// Should have DIFFERENT filenames
	assert.NotEqual(t, resolved1, resolved2, "Different OCI refs should produce different file paths")
	assert.NotEqual(t, filepath.Base(resolved1), filepath.Base(resolved2), "Different OCI refs should produce different filenames")

	content1, err := os.ReadFile(resolved1)
	require.NoError(t, err)
	assert.Equal(t, agent1Content, string(content1))

	content2, err := os.ReadFile(resolved2)
	require.NoError(t, err)
	assert.Equal(t, agent2Content, string(content2))

	assert.Equal(t, OciRefToFilename(ociRef1), filepath.Base(resolved1))
	assert.Equal(t, OciRefToFilename(ociRef2), filepath.Base(resolved2))
}

func TestResolveAgentFile_ContextCancellation(t *testing.T) {
	storeDir := t.TempDir()
	t.Setenv("CAGENT_CONTENT_STORE", storeDir)

	store, err := content.NewStore()
	require.NoError(t, err)

	agentContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Test agent
    instruction: You are a test agent
`
	agentFile := filepath.Join(t.TempDir(), "agent.yaml")
	require.NoError(t, os.WriteFile(agentFile, []byte(agentContent), 0o644))

	// Package as OCI artifact
	ociRef := "test.io/cleanup/agent:v1"
	_, err = oci.PackageFileAsOCIToStore(t.Context(), agentFile, ociRef, store)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())

	// Resolve the OCI ref
	resolved, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)
	assert.FileExists(t, resolved)

	// Cancel the context to trigger cleanup
	cancel()

	// Give the cleanup goroutine time to execute
	time.Sleep(100 * time.Millisecond)

	// File should be deleted
	_, err = os.Stat(resolved)
	assert.True(t, os.IsNotExist(err), "File should be cleaned up after context cancellation")
}

func TestIsOCIReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid OCI references
		{
			name:     "simple repository with tag",
			input:    "myregistry/myrepo:latest",
			expected: true,
		},
		{
			name:     "repository with digest",
			input:    "myregistry/myrepo@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			expected: true,
		},
		{
			name:     "docker hub image",
			input:    "nginx:latest",
			expected: true,
		},
		{
			name:     "fully qualified registry",
			input:    "ghcr.io/docker/cagent:v1.0.0",
			expected: true,
		},
		{
			name:     "registry with port",
			input:    "localhost:5000/myimage:tag",
			expected: true,
		},

		// Local files - NOT OCI references
		{
			name:     "yaml file",
			input:    "agent.yaml",
			expected: false,
		},
		{
			name:     "yml file",
			input:    "config.yml",
			expected: false,
		},
		{
			name:     "yaml file with path",
			input:    "/path/to/agent.yaml",
			expected: false,
		},
		{
			name:     "file descriptor",
			input:    "/dev/fd/3",
			expected: false,
		},

		// Invalid inputs - NOT valid OCI references
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "typo in yaml filename",
			input:    "my-agnt.yaml",
			expected: false,
		},
		{
			name:     "invalid OCI reference with too many colons",
			input:    "invalid:reference:with:too:many:colons",
			expected: false,
		},
		{
			name:     "random string",
			input:    "not-a-valid-reference!!!",
			expected: false,
		},
		{
			name:     "non-existent directory path that looks like OCI ref",
			input:    "/path/to/agents",
			expected: true, // Parses as valid OCI ref if path doesn't exist
		},
		{
			name:     "existing directory",
			input:    t.TempDir(),
			expected: false, // Existing paths are NOT OCI references
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsOCIReference(tt.input)

			assert.Equal(t, tt.expected, result, "isOCIReference(%q) = %v, want %v", tt.input, result, tt.expected)
		})
	}
}

func TestResolveAgentFile_EmptyIsDefault(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve(t.Context(), nil, "")

	require.NoError(t, err)
	assert.Equal(t, "default", resolved)
}

func TestResolveAgentFile_DefaultIsDefault(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve(t.Context(), nil, "default")

	require.NoError(t, err)
	assert.Equal(t, "default", resolved)
}

func TestResolveAgentFile_ReplaceAliasWithActualFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Prepare an aliased file: alias -> [xxx]/pirate.yaml
	wd := t.TempDir()
	aliasedAgentFile := filepath.Join(wd, "pirate.yaml")
	require.NoError(t, os.WriteFile(aliasedAgentFile, []byte(`some config`), 0o644))

	all, err := aliases.Load()
	require.NoError(t, err)
	all.Set("other", "another_file.yaml")
	all.Set("alias", aliasedAgentFile)
	require.NoError(t, all.Save())

	resolved, err := Resolve(t.Context(), &discardOutput{}, "alias")

	require.NoError(t, err)
	assert.Equal(t, aliasedAgentFile, resolved)
}

func TestResolveAgentFile_ReplaceDefaultAliasWithActualFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Prepare an aliased file: alias -> [xxx]/pirate.yaml
	wd := t.TempDir()
	aliasedAgentFile := filepath.Join(wd, "pirate.yaml")
	require.NoError(t, os.WriteFile(aliasedAgentFile, []byte(`some config`), 0o644))

	all, err := aliases.Load()
	require.NoError(t, err)
	all.Set("other", "another_file.yaml")
	all.Set("default", aliasedAgentFile)
	require.NoError(t, all.Save())

	resolved, err := Resolve(t.Context(), &discardOutput{}, "default")

	require.NoError(t, err)
	assert.Equal(t, aliasedAgentFile, resolved)
}

func TestResolveAgentFile_ReplaceEmptyAliasWithActualFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Prepare an aliased file: alias -> [xxx]/pirate.yaml
	wd := t.TempDir()
	aliasedAgentFile := filepath.Join(wd, "pirate.yaml")
	require.NoError(t, os.WriteFile(aliasedAgentFile, []byte(`some config`), 0o644))

	all, err := aliases.Load()
	require.NoError(t, err)
	all.Set("other", "another_file.yaml")
	all.Set("default", aliasedAgentFile)
	require.NoError(t, all.Save())

	resolved, err := Resolve(t.Context(), &discardOutput{}, "")

	require.NoError(t, err)
	assert.Equal(t, aliasedAgentFile, resolved)
}

func TestResolveAgentFile_OCIRef_HasAVersion(t *testing.T) {
	storeDir := t.TempDir()
	t.Setenv("CAGENT_CONTENT_STORE", storeDir)

	store, err := content.NewStore()
	require.NoError(t, err)

	agentContent := `agents:
  root:
    model: auto
    description: Test OCI agent
    instruction: You are a test OCI agent
`
	agentFile := filepath.Join(t.TempDir(), "oci-agent.yaml")
	require.NoError(t, os.WriteFile(agentFile, []byte(agentContent), 0o644))

	// Package as OCI artifact
	ociRef := "test.registry.io/myorg/testagent:v1"
	_, err = oci.PackageFileAsOCIToStore(t.Context(), agentFile, ociRef, store)
	require.NoError(t, err)

	// First resolution
	resolved, err := Resolve(t.Context(), nil, ociRef)
	require.NoError(t, err)
	assert.NotEmpty(t, resolved)

	storedContent, err := os.ReadFile(resolved)
	require.NoError(t, err)
	assert.Equal(t, `version: "`+latest.Version+`"
agents:
  root:
    model: auto
    description: Test OCI agent
    instruction: You are a test OCI agent
`, string(storedContent))
}

// TestResolveAgentFile_OCIRef_CachedUpToDate tests the optimization where
// if local cache matches remote digest, we don't reload from store
func TestResolveAgentFile_OCIRef_CachedUpToDate(t *testing.T) {
	storeDir := t.TempDir()
	t.Setenv("CAGENT_CONTENT_STORE", storeDir)

	store, err := content.NewStore()
	require.NoError(t, err)

	agentContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Cached agent test
    instruction: You are a test agent for cache optimization
`
	agentFile := filepath.Join(t.TempDir(), "cached-agent.yaml")
	require.NoError(t, os.WriteFile(agentFile, []byte(agentContent), 0o644))

	// Package as OCI artifact
	ociRef := "test.registry.io/myorg/cached-agent:v1"
	digest, err := oci.PackageFileAsOCIToStore(t.Context(), agentFile, ociRef, store)
	require.NoError(t, err)
	require.NotEmpty(t, digest, "Digest should not be empty")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// First resolution - should load from store
	resolved1, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)
	assert.NotEmpty(t, resolved1)

	content1, err := os.ReadFile(resolved1)
	require.NoError(t, err)
	assert.Contains(t, string(content1), "Cached agent test")

	// Get the digest from metadata to verify it matches
	meta, err := store.GetArtifactMetadata(ociRef)
	require.NoError(t, err)
	assert.Equal(t, digest, meta.Digest, "Stored digest should match package digest")

	// Second resolution - digest hasn't changed, should use cached version
	// The optimization means we don't call FromStore a second time
	resolved2, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)

	// Should resolve to the same file (deterministic filename from OCI ref)
	assert.Equal(t, resolved1, resolved2)

	content2, err := os.ReadFile(resolved2)
	require.NoError(t, err)
	assert.Equal(t, string(content1), string(content2))
}

// TestResolveAgentFile_OCIRef_ContentUpdated tests that when remote content
// changes (different digest), we reload from store
func TestResolveAgentFile_OCIRef_ContentUpdated(t *testing.T) {
	storeDir := t.TempDir()
	t.Setenv("CAGENT_CONTENT_STORE", storeDir)

	store, err := content.NewStore()
	require.NoError(t, err)

	// Initial content
	agentContent1 := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Original agent
    instruction: Original instruction
`
	agentFile1 := filepath.Join(t.TempDir(), "agent-v1.yaml")
	require.NoError(t, os.WriteFile(agentFile1, []byte(agentContent1), 0o644))

	ociRef := "test.registry.io/myorg/updating-agent:latest"
	digest1, err := oci.PackageFileAsOCIToStore(t.Context(), agentFile1, ociRef, store)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// First resolution
	resolved1, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)

	content1, err := os.ReadFile(resolved1)
	require.NoError(t, err)
	assert.Contains(t, string(content1), "Original agent")

	// Update the content with different data
	agentContent2 := `version: "1"
agents:
  root:
    model: anthropic/claude-sonnet-4-0
    description: Updated agent
    instruction: Updated instruction
`
	agentFile2 := filepath.Join(t.TempDir(), "agent-v2.yaml")
	require.NoError(t, os.WriteFile(agentFile2, []byte(agentContent2), 0o644))

	// Re-package with same OCI ref but different content
	digest2, err := oci.PackageFileAsOCIToStore(t.Context(), agentFile2, ociRef, store)
	require.NoError(t, err)
	assert.NotEqual(t, digest1, digest2, "Digests should differ for different content")

	// Second resolution - should detect digest change and reload
	resolved2, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)

	// Should use same filename (based on OCI ref)
	assert.Equal(t, resolved1, resolved2)

	// But content should be updated
	content2, err := os.ReadFile(resolved2)
	require.NoError(t, err)
	assert.Contains(t, string(content2), "Updated agent")
	assert.NotContains(t, string(content2), "Original agent")
}

// TestResolveAgentFile_OCIRef_DigestComparison verifies digest comparison
// logic by checking metadata before and after resolution
func TestResolveAgentFile_OCIRef_DigestComparison(t *testing.T) {
	storeDir := t.TempDir()
	t.Setenv("CAGENT_CONTENT_STORE", storeDir)

	store, err := content.NewStore()
	require.NoError(t, err)

	agentContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Digest test agent
    instruction: Testing digest logic
`
	agentFile := filepath.Join(t.TempDir(), "digest-agent.yaml")
	require.NoError(t, os.WriteFile(agentFile, []byte(agentContent), 0o644))

	ociRef := "test.registry.io/myorg/digest-test:v1"
	expectedDigest, err := oci.PackageFileAsOCIToStore(t.Context(), agentFile, ociRef, store)
	require.NoError(t, err)

	// Verify metadata exists before resolution
	meta, err := store.GetArtifactMetadata(ociRef)
	require.NoError(t, err)
	assert.Equal(t, expectedDigest, meta.Digest)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Resolve - should compare digests internally
	resolved, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)
	assert.NotEmpty(t, resolved)

	// Verify metadata still matches after resolution
	meta2, err := store.GetArtifactMetadata(ociRef)
	require.NoError(t, err)
	assert.Equal(t, expectedDigest, meta2.Digest)
	assert.Equal(t, meta.Digest, meta2.Digest)

	// Verify content is correct
	fileContent, err := os.ReadFile(resolved)
	require.NoError(t, err)
	assert.Contains(t, string(fileContent), "Digest test agent")
}

// TestResolveAgentFile_OCIRef_NoLocalCache tests the case where no local
// cache exists, so we must load from store after pull
func TestResolveAgentFile_OCIRef_NoLocalCache(t *testing.T) {
	storeDir := t.TempDir()
	t.Setenv("CAGENT_CONTENT_STORE", storeDir)

	store, err := content.NewStore()
	require.NoError(t, err)

	agentContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Fresh pull agent
    instruction: Testing fresh pull
`
	agentFile := filepath.Join(t.TempDir(), "fresh-agent.yaml")
	require.NoError(t, os.WriteFile(agentFile, []byte(agentContent), 0o644))

	// Package in store but don't access it yet
	ociRef := "test.registry.io/myorg/fresh-agent:v1"
	_, err = oci.PackageFileAsOCIToStore(t.Context(), agentFile, ociRef, store)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Resolution should work even though we haven't called FromStore manually yet
	resolved, err := Resolve(ctx, nil, ociRef)
	require.NoError(t, err)
	assert.NotEmpty(t, resolved)

	// Content should be correct
	fileContent, err := os.ReadFile(resolved)
	require.NoError(t, err)
	assert.Contains(t, string(fileContent), "Fresh pull agent")
}

func TestResolveAgentFile_Directory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Should resolve directory to absolute path
	resolved, err := Resolve(ctx, nil, tmpDir)
	require.NoError(t, err)

	absPath, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, absPath, resolved)
}

func TestResolveAgentFile_DirectoryWithAgentYaml(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Test agent in directory
    instruction: You are a test agent
`
	agentFile := filepath.Join(tmpDir, "agent.yaml")
	require.NoError(t, os.WriteFile(agentFile, []byte(agentContent), 0o644))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Resolving directory should return directory path
	resolved, err := Resolve(ctx, nil, tmpDir)
	require.NoError(t, err)

	absPath, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, absPath, resolved)
}

func TestResolveAgentFile_RelativeDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Test agent
    instruction: You are a test agent
`
	agentFile := filepath.Join(tmpDir, "agent.yaml")
	require.NoError(t, os.WriteFile(agentFile, []byte(agentContent), 0o644))

	// Change to temp directory and use relative path
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	require.NoError(t, os.Chdir(tmpDir))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Resolve relative path "."
	resolved, err := Resolve(ctx, nil, ".")
	require.NoError(t, err)

	absPath, err := filepath.Abs(".")
	require.NoError(t, err)
	assert.Equal(t, absPath, resolved)
}

func TestResolveAgentFile_NonExistentDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "does-not-exist")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Should try to treat as OCI reference and fail
	_, err := Resolve(ctx, nil, nonExistentDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to pull OCI image")
}

func TestResolveAgentFile_DirectoryVsFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a directory
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0o755))

	// Create a file with same name but .yaml extension
	yamlFile := filepath.Join(tmpDir, "subdir.yaml")
	yamlContent := `version: "1"
agents:
  root:
    model: openai/gpt-4o
    description: Test agent file
    instruction: You are a test agent
`
	require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0o644))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Resolve directory
	resolvedDir, err := Resolve(ctx, nil, subDir)
	require.NoError(t, err)
	absDirPath, err := filepath.Abs(subDir)
	require.NoError(t, err)
	assert.Equal(t, absDirPath, resolvedDir)

	// Resolve file
	resolvedFile, err := Resolve(ctx, nil, yamlFile)
	require.NoError(t, err)
	absFilePath, err := filepath.Abs(yamlFile)
	require.NoError(t, err)
	assert.Equal(t, absFilePath, resolvedFile)

	// They should be different paths
	assert.NotEqual(t, resolvedDir, resolvedFile)
}

func TestResolveAgentFile_EmptyDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.Mkdir(emptyDir, 0o755))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Should still resolve the directory path
	resolved, err := Resolve(ctx, nil, emptyDir)
	require.NoError(t, err)

	absPath, err := filepath.Abs(emptyDir)
	require.NoError(t, err)
	assert.Equal(t, absPath, resolved)
}

type discardOutput struct{}

func (d *discardOutput) Printf(string, ...any) {}
