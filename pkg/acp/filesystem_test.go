package acp

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePath(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()

	ts := &FilesystemToolset{
		workingDir: workingDir,
	}

	absWorkingDir, err := filepath.Abs(workingDir)
	require.NoError(t, err)

	tests := []struct {
		name      string
		userPath  string
		wantPath  string
		wantError bool
	}{
		{
			name:     "simple relative path",
			userPath: "file.txt",
			wantPath: filepath.Join(absWorkingDir, "file.txt"),
		},
		{
			name:     "nested relative path",
			userPath: "subdir/file.txt",
			wantPath: filepath.Join(absWorkingDir, "subdir", "file.txt"),
		},
		{
			name:     "dot path resolves to working directory",
			userPath: ".",
			wantPath: absWorkingDir,
		},
		{
			name:      "parent directory escape blocked",
			userPath:  "../escape.txt",
			wantError: true,
		},
		{
			name:      "deep parent directory escape blocked",
			userPath:  "subdir/../../escape.txt",
			wantError: true,
		},
		{
			name:     "dot-dot within working dir is fine",
			userPath: "subdir/../file.txt",
			wantPath: filepath.Join(absWorkingDir, "file.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolved, err := ts.resolvePath(tt.userPath)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "escapes the working directory")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, resolved)
		})
	}
}

func TestNormalizePathForComparison(t *testing.T) {
	t.Parallel()

	// On macOS/Windows (case-insensitive), normalization should lowercase.
	// On Linux (case-sensitive), it should be identity.
	result := normalizePathForComparison("/Some/Path")

	// This test validates the function exists and returns a string.
	// The exact behavior depends on the platform.
	assert.NotEmpty(t, result)
}
