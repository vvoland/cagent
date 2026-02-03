package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/cmd/root"
	"github.com/docker/cagent/pkg/server"
)

type Session struct {
	Title string `json:"title"`
}

func TestCagentAPI_ListSessions(t *testing.T) {
	type testcase struct {
		db            string
		expectedCount int
	}

	for _, tc := range []testcase{
		{"one-session.db", 1},
		{"two-sessions.db", 2},
		{"transfer-task.db", 1},
		{"session.db", 1},
		{"session-not-found.db", 17},
		{"desktop.db", 2},
	} {
		t.Run(tc.db, func(t *testing.T) {
			dbPath, err := filepath.Abs("testdata/db/" + tc.db)
			require.NoError(t, err)

			socketPath := startCagentAPI(t, dbPath)
			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
						return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
					},
				},
			}

			resp, err := client.Get("http://localhost/api/sessions")
			require.NoError(t, err)
			defer resp.Body.Close()

			var sessions []Session
			err = json.NewDecoder(resp.Body).Decode(&sessions)
			require.NoError(t, err)

			assert.Len(t, sessions, tc.expectedCount)
		})
	}
}

func startCagentAPI(t *testing.T, db string) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	copyFile(t, "session.db", db)
	if _, err := os.Stat(db + "-wal"); err == nil {
		copyFile(t, "session.db-wal", db+"-wal")
	}

	ln, err := server.Listen(t.Context(), "unix://cagent.sock")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = ln.Close()
	})

	file, err := ln.(*net.UnixListener).File()
	require.NoError(t, err)

	go func() {
		var stdout, stderr bytes.Buffer
		_ = root.Execute(t.Context(), nil, &stdout, &stderr, "api", "-s", "session.db", "--listen", fmt.Sprintf("fd://%d", file.Fd()), "default")
	}()

	return "cagent.sock"
}

func copyFile(t *testing.T, dst, src string) {
	t.Helper()

	srcFile, err := os.Open(src)
	require.NoError(t, err)
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	require.NoError(t, err)
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	require.NoError(t, err)
}
