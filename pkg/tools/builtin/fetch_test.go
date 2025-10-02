package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/tools"
)

func TestFetchToolWithOptions(t *testing.T) {
	customTimeout := 60 * time.Second
	tool := NewFetchTool(WithTimeout(customTimeout))

	require.Equal(t, customTimeout, tool.handler.timeout)
}

func TestFetchTool_Tools(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.TODO()

	toolSet, err := tool.Tools(ctx)
	require.NoError(t, err)
	require.Len(t, toolSet, 1)

	fetchTool := toolSet[0]
	require.Equal(t, "fetch", fetchTool.Function.Name)
	require.NotNil(t, fetchTool.Handler)
}

func TestFetchTool_Instructions(t *testing.T) {
	tool := NewFetchTool()
	instructions := tool.Instructions()

	require.NotEmpty(t, instructions)

	require.Contains(t, instructions, `"fetch" tool instructions`)
	require.Contains(t, instructions, "HTTP")
	require.Contains(t, instructions, "HTTPS")
	require.Contains(t, instructions, "URLs")
}

func TestFetchTool_StartStop(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.TODO()

	err := tool.Start(ctx)
	require.NoError(t, err)

	err = tool.Stop()
	require.NoError(t, err)
}

func TestFetchHandler_CallTool_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Hello, World!")
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx := context.TODO()

	args := map[string]any{
		"urls": []string{server.URL},
	}
	argsJSON, _ := json.Marshal(args)

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: string(argsJSON),
		},
	}

	result, err := tool.handler.CallTool(ctx, toolCall)
	require.NoError(t, err)

	require.NotNil(t, result)

	require.Contains(t, result.Output, "Successfully fetched")
	require.Contains(t, result.Output, "Status: 200")
	require.Contains(t, result.Output, "Length: 13 bytes")
	require.Contains(t, result.Output, "Hello, World!")
}

func TestFetchHandler_CallTool_MultipleURLs(t *testing.T) {
	// Create test servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Server 1")
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Server 2")
	}))
	defer server2.Close()

	tool := NewFetchTool()
	ctx := context.TODO()

	args := map[string]any{
		"urls": []string{server1.URL, server2.URL},
	}
	argsJSON, _ := json.Marshal(args)

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: string(argsJSON),
		},
	}

	result, err := tool.handler.CallTool(ctx, toolCall)
	require.NoError(t, err)

	// Should return JSON for multiple URLs
	var results []FetchResult
	err = json.Unmarshal([]byte(result.Output), &results)
	require.NoError(t, err)
	require.Len(t, results, 2)

	require.Equal(t, "Server 1", results[0].Body)
	require.Equal(t, "Server 2", results[1].Body)
}

func TestFetchHandler_CallTool_InvalidURL(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.TODO()

	args := map[string]any{
		"urls": []string{"not-a-url"},
	}
	argsJSON, _ := json.Marshal(args)

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: string(argsJSON),
		},
	}

	result, err := tool.handler.CallTool(ctx, toolCall)
	require.NoError(t, err)
	require.Contains(t, result.Output, "Error fetching")
}

func TestFetchHandler_CallTool_UnsupportedProtocol(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.TODO()

	args := map[string]any{
		"urls": []string{"ftp://example.com"},
	}
	argsJSON, _ := json.Marshal(args)

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: string(argsJSON),
		},
	}

	result, err := tool.handler.CallTool(ctx, toolCall)
	require.NoError(t, err)

	require.Contains(t, result.Output, "Error fetching")
	require.Contains(t, result.Output, "only HTTP and HTTPS URLs are supported")
}

func TestFetchHandler_CallTool_NoURLs(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.TODO()

	args := map[string]any{
		"urls": []string{},
	}
	argsJSON, _ := json.Marshal(args)

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: string(argsJSON),
		},
	}

	_, err := tool.handler.CallTool(ctx, toolCall)
	require.Error(t, err)

	require.Equal(t, "at least one URL is required", err.Error())
}

func TestFetchHandler_CallTool_InvalidJSON(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.TODO()

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: "invalid json",
		},
	}

	_, err := tool.handler.CallTool(ctx, toolCall)
	require.Error(t, err)
}

func TestFetchHandler_CallTool_CustomMethod(t *testing.T) {
	// Create test server that checks method
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		fmt.Fprint(w, "POST received")
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx := context.TODO()

	args := map[string]any{
		"urls":   []string{server.URL},
		"method": "POST",
	}
	argsJSON, _ := json.Marshal(args)

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: string(argsJSON),
		},
	}

	result, err := tool.handler.CallTool(ctx, toolCall)
	require.NoError(t, err)

	require.Contains(t, result.Output, "Successfully fetched")
}

func TestFetchHandler_Markdown(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<h1>Hello cagent</h1>")
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx := context.TODO()

	args := map[string]any{
		"urls":   []string{server.URL},
		"format": "markdown",
	}
	argsJSON, _ := json.Marshal(args)

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: string(argsJSON),
		},
	}

	result, err := tool.handler.CallTool(ctx, toolCall)
	require.NoError(t, err)

	require.NotNil(t, result)

	require.Contains(t, result.Output, "Successfully fetched")
	require.Contains(t, result.Output, "Status: 200")
	require.Contains(t, result.Output, "Length: 14 bytes")
	require.Contains(t, result.Output, "# Hello cagent")
}

func TestFetchHandler_Text(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<h1>Hello cagent</h1>")
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx := context.TODO()

	args := map[string]any{
		"urls":   []string{server.URL},
		"format": "text",
	}
	argsJSON, _ := json.Marshal(args)

	toolCall := tools.ToolCall{
		Function: tools.FunctionCall{
			Arguments: string(argsJSON),
		},
	}

	result, err := tool.handler.CallTool(ctx, toolCall)
	require.NoError(t, err)

	require.NotNil(t, result)

	require.Contains(t, result.Output, "Successfully fetched")
	require.Contains(t, result.Output, "Status: 200")
	require.Contains(t, result.Output, "Length: 12 bytes")
	require.Contains(t, result.Output, "Hello cagent")
}
