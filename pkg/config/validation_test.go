package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig_InvalidPath(t *testing.T) {
	tmp := openRoot(t, t.TempDir())

	validConfig := `version: 1
agents:
  root:
    model: "openai/gpt-4"
`

	err := tmp.WriteFile("valid.yaml", []byte(validConfig), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(t.Context(), "valid.yaml", tmp)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	_, err = LoadConfig(t.Context(), "../../../etc/passwd", tmp)
	require.Error(t, err)
}

func TestValidationErrors(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := openRoot(t, "testdata")

			_, err := LoadConfig(t.Context(), tt.path, root)
			require.Error(t, err)
		})
	}
}
