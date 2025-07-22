package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnv(t *testing.T) {
	env := []string{"USER=alice", "HOME=/home/alice"}

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello $USER", "Hello alice"},
		{"Your home is at $HOME", "Your home is at /home/alice"},
		{"No variable here", "No variable here"},
		{"$UNKNOWN_VAR should be empty", " should be empty"},
	}

	for _, test := range tests {
		result := expandEnv(test.input, env)

		assert.Equal(t, test.expected, result)
	}
}

func TestExpandTildePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "no tilde",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "tilde only",
			input:    "~",
			expected: homeDir,
		},
		{
			name:     "tilde with slash",
			input:    "~/env/slack.env",
			expected: filepath.Join(homeDir, "env/slack.env"),
		},
		{
			name:     "tilde with deeper path",
			input:    "~/config/app/settings.env",
			expected: filepath.Join(homeDir, "config/app/settings.env"),
		},
		{
			name:        "unsupported tilde format",
			input:       "~user/path",
			expected:    "",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := expandTildePath(test.input)
			if test.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported tilde expansion format")
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestReadEnvFilesEmpty(t *testing.T) {
	lines, err := readEnvFiles("/some/dir", []string{})

	require.NoError(t, err)
	assert.Empty(t, lines)
}

func TestReadEnvFiles(t *testing.T) {
	temp := t.TempDir()
	write(t, filepath.Join(temp, ".env1"), "KEY1=VALUE1\n# Comment\nKEY2=VALUE2\n")
	write(t, filepath.Join(temp, ".env2"), "\n\nKEY3=VALUE3\n")

	lines, err := readEnvFiles(temp, []string{".env1", ".env2"})

	require.NoError(t, err)
	assert.Len(t, lines, 3)
	assert.Equal(t, "KEY1", lines[0].Key)
	assert.Equal(t, "VALUE1", lines[0].Value)
	assert.Equal(t, "KEY2", lines[1].Key)
	assert.Equal(t, "VALUE2", lines[1].Value)
	assert.Equal(t, "KEY3", lines[2].Key)
	assert.Equal(t, "VALUE3", lines[2].Value)
}

func TestReadEnvFileWithTildePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	// Create a temporary file in a subdirectory of the home directory
	testDir := filepath.Join(homeDir, "test-cagent-env")
	err = os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	envFile := filepath.Join(testDir, "test.env")
	write(t, envFile, "TILDE_KEY=TILDE_VALUE\n")

	// Test with tilde path
	tildePath := "~/test-cagent-env/test.env"
	lines, err := readEnvFile("/some/parent/dir", tildePath)

	require.NoError(t, err)
	assert.Len(t, lines, 1)
	assert.Equal(t, "TILDE_KEY", lines[0].Key)
	assert.Equal(t, "TILDE_VALUE", lines[0].Value)
}

func TestReadEnvFileNotFound(t *testing.T) {
	temp := t.TempDir()

	lines, err := readEnvFiles(temp, []string{".notfound"})

	require.Error(t, err)
	assert.Empty(t, lines)
}

func TestReadEnvFileInvalid(t *testing.T) {
	temp := t.TempDir()
	write(t, filepath.Join(temp, ".invalid"), "The is not a valid env file")

	lines, err := readEnvFiles(temp, []string{".invalid"})

	require.Error(t, err)
	assert.Empty(t, lines)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}
