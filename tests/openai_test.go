package tests

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/cmd/root"
)

func TestOpenAI_SimpleResponse(t *testing.T) {
	t.Parallel()

	dotEnv := filepath.Join(t.TempDir(), ".env")
	err := os.WriteFile(dotEnv, []byte("DOCKER_TOKEN=DUMMY"), 0o644)
	require.NoError(t, err)

	ctx := t.Context()
	svr := startFakeOpenAIServer(t,
		WithResponseForQuestion("How are you doing?", "Good!"),
	)

	var stdout bytes.Buffer
	err = root.Execute(ctx, nil, &stdout, io.Discard, "exec", "testdata/basic.yaml", "--models-gateway", svr.URL, "--env-from-file", dotEnv, "How are you doing?")

	require.NoError(t, err)
	require.Equal(t, "\n--- Agent: root ---\n\nGood!", stdout.String())
}
