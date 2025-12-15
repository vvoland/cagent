package e2e_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/cmd/root"
)

func TestExec_Record_CreatesCassette(t *testing.T) {
	tmpDir := t.TempDir()
	cassettePath := filepath.Join(tmpDir, "test-recording")

	dotEnv := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(dotEnv, []byte("DOCKER_TOKEN=DUMMY"), 0o644)
	require.NoError(t, err)

	args := []string{
		"exec",
		"--env-from-file", dotEnv,
		"--session-db", filepath.Join(tmpDir, "session.db"),
		"--record=" + cassettePath,
		"testdata/basic.yaml",
		"What's 2+2?",
	}

	var stdout bytes.Buffer
	err = root.Execute(t.Context(), nil, &stdout, io.Discard, args...)
	require.NoError(t, err)

	cassetteFile := cassettePath + ".yaml"
	_, err = os.Stat(cassetteFile)
	require.NoError(t, err, "cassette file should be created at %s", cassetteFile)

	content, err := os.ReadFile(cassetteFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "interactions:", "cassette should contain interactions")
	assert.Contains(t, string(content), "api.anthropic.com", "cassette should contain API host")
}

func TestExec_Record_AutoGeneratesFilename(t *testing.T) {
	tmpDir := t.TempDir()

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})

	dotEnv := filepath.Join(tmpDir, ".env")
	err = os.WriteFile(dotEnv, []byte("DOCKER_TOKEN=DUMMY"), 0o644)
	require.NoError(t, err)

	args := []string{
		"exec",
		"--env-from-file", dotEnv,
		"--session-db", filepath.Join(tmpDir, "session.db"),
		"--record", // No value = auto-generate
		originalWd + "/testdata/basic.yaml",
		"What's 2+2?",
	}

	var stdout bytes.Buffer
	err = root.Execute(t.Context(), nil, &stdout, io.Discard, args...)
	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(tmpDir, "cagent-recording-*.yaml"))
	require.NoError(t, err)
	assert.Len(t, files, 1, "should have created exactly one auto-generated cassette file")

	content, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Contains(t, string(content), "interactions:", "cassette should contain interactions")
}

func TestAPI_FakeAndRecord_MutuallyExclusive(t *testing.T) {
	args := []string{
		"api",
		"--fake=test.yaml",
		"--record=output.yaml",
		"testdata/basic.yaml",
	}

	var stdout, stderr bytes.Buffer
	err := root.Execute(t.Context(), nil, &stdout, &stderr, args...)

	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "fake") && strings.Contains(err.Error(), "record"),
		"error should mention both flags: %v", err)
}
