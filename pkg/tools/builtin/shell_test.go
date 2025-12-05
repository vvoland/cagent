package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/tools"
)

func TestNewShellTool(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	tool := NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})

	assert.NotNil(t, tool)
	assert.NotNil(t, tool.handler)
	assert.Equal(t, "/bin/bash", tool.handler.shell)

	t.Setenv("SHELL", "")
	tool = NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})

	assert.NotNil(t, tool)
	assert.NotNil(t, tool.handler)
	assert.Equal(t, "/bin/sh", tool.handler.shell, "Should default to /bin/sh when SHELL is not set")
}

func TestShellTool_HandlerEcho(t *testing.T) {
	tool := NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})

	result, err := tool.handler.RunShell(t.Context(), RunShellArgs{
		Cmd: "echo 'hello world'",
		Cwd: "",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "hello world")
}

func TestShellTool_HandlerWithCwd(t *testing.T) {
	tool := NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})
	tmpDir := t.TempDir()

	result, err := tool.handler.RunShell(t.Context(), RunShellArgs{
		Cmd: "pwd",
		Cwd: tmpDir,
	})
	require.NoError(t, err)
	// The output might contain extra newlines or other characters,
	// so we just check if it contains the temp dir path
	assert.Contains(t, result.Output, tmpDir)
}

func TestShellTool_HandlerError(t *testing.T) {
	tool := NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})

	result, err := tool.handler.RunShell(t.Context(), RunShellArgs{
		Cmd: "command_that_does_not_exist",
		Cwd: "",
	})
	require.NoError(t, err, "Handler should not return an error")
	assert.Contains(t, result.Output, "Error executing command")
}

func TestShellTool_OutputSchema(t *testing.T) {
	tool := NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})

	allTools, err := tool.Tools(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, allTools)

	for _, tool := range allTools {
		assert.NotNil(t, tool.OutputSchema)
	}
}

func TestShellTool_ParametersAreObjects(t *testing.T) {
	tool := NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})

	allTools, err := tool.Tools(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, allTools)

	for _, tool := range allTools {
		m, err := tools.SchemaToMap(tool.Parameters)
		require.NoError(t, err)
		assert.Equal(t, "object", m["type"])
	}
}

// Minimal tests for background job features
func TestShellTool_RunBackgroundJob(t *testing.T) {
	tool := NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})
	err := tool.Start(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tool.Stop(t.Context())
	})

	result, err := tool.handler.RunShellBackground(t.Context(), RunShellBackgroundArgs{Cmd: "echo test"})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "Background job started with ID:")
}

func TestShellTool_ListBackgroundJobs(t *testing.T) {
	tool := NewShellTool(nil, &config.RuntimeConfig{Config: config.Config{WorkingDir: t.TempDir()}})
	err := tool.Start(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tool.Stop(t.Context())
	})

	// Start a background job first
	_, err = tool.handler.RunShellBackground(t.Context(), RunShellBackgroundArgs{Cmd: "echo test"})
	require.NoError(t, err)

	// No need to wait - ListBackgroundJobs shows jobs regardless of status
	listResult, err := tool.handler.ListBackgroundJobs(t.Context(), nil)

	require.NoError(t, err)
	assert.Contains(t, listResult.Output, "Background Jobs:")
	assert.Contains(t, listResult.Output, "ID: job_")
}
