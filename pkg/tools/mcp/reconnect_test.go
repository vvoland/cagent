package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker-agent/pkg/tools"
)

// TestRemoteReconnectAfterServerRestart verifies that a Toolset backed by a
// real remote (streamable-HTTP) MCP server transparently recovers when the
// server is restarted.
//
// The scenario:
//  1. Start a minimal MCP server with a "ping" tool.
//  2. Connect a Toolset, call "ping" — succeeds.
//  3. Shut down the server (simulates crash / restart).
//  4. Start a **new** server on the same address.
//  5. Call "ping" again — this must succeed after automatic reconnection.
//
// Without the ErrSessionMissing recovery logic the second call would fail
// because the new server does not know the old session ID.
func TestRemoteReconnectAfterServerRestart(t *testing.T) {
	t.Parallel()

	// Use a fixed listener address so we can restart on the same port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close() // We only needed the address; close so startServer can bind it.

	var callCount atomic.Int32

	// startServer creates a minimal MCP server on addr with a "ping" tool
	// and returns a function to shut it down.
	startServer := func(t *testing.T) (shutdown func()) {
		t.Helper()

		s := gomcp.NewServer(&gomcp.Implementation{Name: "test-server", Version: "1.0.0"}, nil)
		s.AddTool(&gomcp.Tool{
			Name:        "ping",
			InputSchema: &jsonschema.Schema{Type: "object"},
		}, func(_ context.Context, _ *gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
			n := callCount.Add(1)
			return &gomcp.CallToolResult{
				Content: []gomcp.Content{&gomcp.TextContent{Text: fmt.Sprintf("pong-%d", n)}},
			}, nil
		})

		// Retry Listen until the port is available (e.g. after a server shutdown).
		var srvLn net.Listener
		require.Eventually(t, func() bool {
			var listenErr error
			srvLn, listenErr = net.Listen("tcp", addr)
			return listenErr == nil
		}, 2*time.Second, 50*time.Millisecond, "port %s not available in time", addr)

		srv := &http.Server{
			Handler: gomcp.NewStreamableHTTPHandler(func(*http.Request) *gomcp.Server { return s }, nil),
		}
		go func() { _ = srv.Serve(srvLn) }()

		return func() { _ = srv.Close() }
	}

	callPing := func(t *testing.T, ts *Toolset) string {
		t.Helper()
		result, callErr := ts.callTool(t.Context(), tools.ToolCall{
			Function: tools.FunctionCall{Name: "ping", Arguments: "{}"},
		})
		require.NoError(t, callErr)
		return result.Output
	}

	// --- Step 1–2: Start first server, connect toolset ---
	shutdown1 := startServer(t)

	ts := NewRemoteToolset("test", fmt.Sprintf("http://%s/mcp", addr), "streamable-http", nil)
	require.NoError(t, ts.Start(t.Context()))

	toolList, err := ts.Tools(t.Context())
	require.NoError(t, err)
	require.Len(t, toolList, 1)
	assert.Equal(t, "test_ping", toolList[0].Name)

	// --- Step 3: Call succeeds on original server ---
	assert.Equal(t, "pong-1", callPing(t, ts))

	// --- Step 4: Shut down the server ---
	shutdown1()

	// Capture the current restarted channel before the reconnect
	ts.mu.Lock()
	restartedCh := ts.restarted
	ts.mu.Unlock()

	// --- Step 5–6: Start a fresh server, call again ---
	shutdown2 := startServer(t)
	t.Cleanup(func() {
		_ = ts.Stop(t.Context())
		shutdown2()
	})

	// This call triggers ErrSessionMissing recovery and must succeed transparently.
	assert.Equal(t, "pong-2", callPing(t, ts))

	// Verify that watchConnection actually restarted the connection by checking
	// that the restarted channel was closed (signaling reconnect completion).
	select {
	case <-restartedCh:
		// Success: the channel was closed, meaning reconnect happened
	case <-time.After(100 * time.Millisecond):
		t.Fatal("reconnect did not complete: restarted channel was not closed")
	}
}
