package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig_InvalidPath(t *testing.T) {
	tmp := t.TempDir()
	tmpRoot := openRoot(t, tmp)

	validConfig := `version: 1
agents:
  root:
    model: "openai/gpt-4"
`

	err := tmpRoot.WriteFile("valid.yaml", []byte(validConfig), 0o644)
	require.NoError(t, err)

	cfg, err := Load(t.Context(), testfileSource(filepath.Join(tmp, "valid.yaml")))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	_, err = Load(t.Context(), testfileSource(filepath.Join(tmp, "../../../etc/passwd"))) //nolint: gocritic // testing invalid path
	require.Error(t, err)
}

func TestValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "memory toolset missing path",
			path: "missing_memory_path_v2.yaml",
		},
		{
			name: "path in non memory toolset",
			path: "invalid_path_v2.yaml",
		},
		{
			name: "post_edit in non filesystem toolset",
			path: "invalid_post_edit_v2.yaml",
		},
		{
			name: "skills enabled without filesystem toolset",
			path: "skills_missing_filesystem.yaml",
		},
		{
			name: "skills enabled without read_file tool",
			path: "skills_missing_read_file.yaml",
		},
		{
			name: "lsp toolset missing command",
			path: "invalid_lsp_missing_command.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Load(t.Context(), testfileSource(filepath.Join("testdata", tt.path)))
			require.Error(t, err)
		})
	}
}

func TestValidSkillsConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "skills with all filesystem tools",
			path: "skills_valid_all_tools.yaml",
		},
		{
			name: "skills with explicit read_file tool",
			path: "skills_valid_explicit_tools.yaml",
		},
		{
			name: "skills disabled",
			path: "skills_disabled.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := Load(t.Context(), testfileSource(filepath.Join("testdata", tt.path)))
			require.NoError(t, err)
			require.NotNil(t, cfg)
		})
	}
}
