package root

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigShowCommand_Empty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"show"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Empty config outputs as empty YAML object
	output := buf.String()
	assert.Equal(t, "{}\n", output)
}

func TestConfigShowCommand_WithAliases(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create config directory and file
	configDir := filepath.Join(home, ".config", "cagent")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configContent := `aliases:
  code:
    path: agentcatalog/coder
  docs:
    path: agentcatalog/docs-writer
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644))

	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"show"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "aliases:")
	assert.Contains(t, output, "code:")
	assert.Contains(t, output, "agentcatalog/coder")
	assert.Contains(t, output, "docs:")
	assert.Contains(t, output, "agentcatalog/docs-writer")
}

func TestConfigShowCommand_DefaultBehavior(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Running "config" without subcommand should default to "show"
	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	// Empty config outputs as empty YAML object
	output := buf.String()
	assert.Equal(t, "{}\n", output)
}

func TestConfigPathCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"path"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, ".config")
	assert.Contains(t, output, "cagent")
	assert.Contains(t, output, "config.yaml")
}

func TestConfigShowCommand_MalformedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create malformed config file
	configDir := filepath.Join(home, ".config", "cagent")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("not: valid: yaml: content"), 0o644))

	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"show"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}
