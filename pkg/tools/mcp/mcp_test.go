package mcp

import (
	"context"
	"iter"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/tools"
)

// mockMCPClient is a test double for the mcpClient interface.
type mockMCPClient struct {
	callToolFn func(ctx context.Context, request *mcp.CallToolParams) (*mcp.CallToolResult, error)
}

func (m *mockMCPClient) Initialize(context.Context, *mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return &mcp.InitializeResult{}, nil
}

func (m *mockMCPClient) ListTools(context.Context, *mcp.ListToolsParams) iter.Seq2[*mcp.Tool, error] {
	return func(func(*mcp.Tool, error) bool) {}
}

func (m *mockMCPClient) CallTool(ctx context.Context, request *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	return m.callToolFn(ctx, request)
}

func (m *mockMCPClient) ListPrompts(context.Context, *mcp.ListPromptsParams) iter.Seq2[*mcp.Prompt, error] {
	return func(func(*mcp.Prompt, error) bool) {}
}

func (m *mockMCPClient) GetPrompt(context.Context, *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{}, nil
}

func (m *mockMCPClient) SetElicitationHandler(tools.ElicitationHandler) {}

func (m *mockMCPClient) SetOAuthSuccessHandler(func()) {}

func (m *mockMCPClient) SetManagedOAuth(bool) {}

func (m *mockMCPClient) SetToolListChangedHandler(func()) {}

func (m *mockMCPClient) SetPromptListChangedHandler(func()) {}

func (m *mockMCPClient) Wait() error { return nil }

func (m *mockMCPClient) Close(context.Context) error { return nil }

func TestCallToolStripsNullArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		arguments    string
		expectedArgs map[string]any
	}{
		{
			name:         "all null values are stripped",
			arguments:    `{"dir": null, "pattern": null}`,
			expectedArgs: map[string]any{},
		},
		{
			name:         "only null values are stripped",
			arguments:    `{"dir": ".", "pattern": null, "extra": "value"}`,
			expectedArgs: map[string]any{"dir": ".", "extra": "value"},
		},
		{
			name:         "empty arguments stay empty",
			arguments:    `{}`,
			expectedArgs: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedArgs map[string]any

			ts := &Toolset{
				started: true,
				mcpClient: &mockMCPClient{
					callToolFn: func(_ context.Context, request *mcp.CallToolParams) (*mcp.CallToolResult, error) {
						if m, ok := request.Arguments.(map[string]any); ok {
							capturedArgs = m
						}
						return &mcp.CallToolResult{
							Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
						}, nil
					},
				},
			}

			result, err := ts.callTool(t.Context(), tools.ToolCall{
				Function: tools.FunctionCall{
					Name:      "test_tool",
					Arguments: tt.arguments,
				},
			})

			require.NoError(t, err)
			assert.Equal(t, "ok", result.Output)
			assert.Equal(t, tt.expectedArgs, capturedArgs)
		})
	}
}

func TestProcessMCPContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          *mcp.CallToolResult
		wantOutput     string
		wantIsError    bool
		wantImages     []tools.MediaContent
		wantAudios     []tools.MediaContent
		wantStructured any
	}{
		// --- text ---
		{
			name:       "text only",
			input:      callToolResult(&mcp.TextContent{Text: "hello"}),
			wantOutput: "hello",
		},
		{
			name:       "empty response",
			input:      &mcp.CallToolResult{},
			wantOutput: "no output",
		},

		// --- images ---
		{
			name:       "image only",
			input:      callToolResult(&mcp.ImageContent{Data: []byte("imagedata"), MIMEType: "image/png"}),
			wantOutput: "no output",
			wantImages: []tools.MediaContent{{Data: "aW1hZ2VkYXRh", MimeType: "image/png"}},
		},
		{
			name:       "text and image",
			input:      callToolResult(&mcp.TextContent{Text: "Here is the screenshot"}, &mcp.ImageContent{Data: []byte("screenshot"), MIMEType: "image/jpeg"}),
			wantOutput: "Here is the screenshot",
			wantImages: []tools.MediaContent{{Data: "c2NyZWVuc2hvdA==", MimeType: "image/jpeg"}},
		},
		{
			name:        "error with image",
			input:       &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "error occurred"}, &mcp.ImageContent{Data: []byte("error"), MIMEType: "image/png"}}},
			wantOutput:  "error occurred",
			wantIsError: true,
			wantImages:  []tools.MediaContent{{Data: "ZXJyb3I=", MimeType: "image/png"}},
		},

		// --- audio ---
		{
			name:       "audio only",
			input:      callToolResult(&mcp.AudioContent{Data: []byte("audiodata"), MIMEType: "audio/wav"}),
			wantOutput: "no output",
			wantAudios: []tools.MediaContent{{Data: "YXVkaW9kYXRh", MimeType: "audio/wav"}},
		},
		{
			name:       "text and audio",
			input:      callToolResult(&mcp.TextContent{Text: "Here is the recording"}, &mcp.AudioContent{Data: []byte("recording"), MIMEType: "audio/mp3"}),
			wantOutput: "Here is the recording",
			wantAudios: []tools.MediaContent{{Data: "cmVjb3JkaW5n", MimeType: "audio/mp3"}},
		},
		{
			name:       "text image and audio",
			input:      callToolResult(&mcp.TextContent{Text: "multimedia"}, &mcp.ImageContent{Data: []byte("img"), MIMEType: "image/png"}, &mcp.AudioContent{Data: []byte("aud"), MIMEType: "audio/wav"}),
			wantOutput: "multimedia",
			wantImages: []tools.MediaContent{{Data: "aW1n", MimeType: "image/png"}},
			wantAudios: []tools.MediaContent{{Data: "YXVk", MimeType: "audio/wav"}},
		},

		// --- resource links ---
		{
			name:       "resource link with name",
			input:      callToolResult(&mcp.ResourceLink{Name: "my-doc", URI: "file:///path/to/doc.txt"}),
			wantOutput: "[my-doc](file:///path/to/doc.txt)",
		},
		{
			name:       "resource link without name",
			input:      callToolResult(&mcp.ResourceLink{URI: "file:///path/to/doc.txt"}),
			wantOutput: "file:///path/to/doc.txt",
		},
		{
			name:       "text and resource link",
			input:      callToolResult(&mcp.TextContent{Text: "See: "}, &mcp.ResourceLink{Name: "readme", URI: "file:///README.md"}),
			wantOutput: "See: [readme](file:///README.md)",
		},
		{
			name:       "resource link name with bracket is escaped",
			input:      callToolResult(&mcp.ResourceLink{Name: "doc]name", URI: "file:///doc.txt"}),
			wantOutput: `[doc\]name](file:///doc.txt)`,
		},
		{
			name:       "resource link URI with paren is escaped",
			input:      callToolResult(&mcp.ResourceLink{Name: "doc", URI: "file:///path(1)/doc.txt"}),
			wantOutput: "[doc](file:///path(1%29/doc.txt)",
		},

		// --- structured content ---
		{
			name:           "structured content passed through",
			input:          &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "done"}}, StructuredContent: map[string]any{"status": "ok", "count": float64(42)}},
			wantOutput:     "done",
			wantStructured: map[string]any{"status": "ok", "count": float64(42)},
		},
		{
			name:       "nil structured content",
			input:      callToolResult(&mcp.TextContent{Text: "hello"}),
			wantOutput: "hello",
		},
		{
			name:           "structured content without text",
			input:          &mcp.CallToolResult{StructuredContent: map[string]any{"key": "value"}},
			wantOutput:     "no output",
			wantStructured: map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := processMCPContent(tt.input)

			assert.Equal(t, tt.wantOutput, result.Output)
			assert.Equal(t, tt.wantIsError, result.IsError)

			if tt.wantImages != nil {
				assert.Equal(t, tt.wantImages, result.Images)
			} else {
				assert.Empty(t, result.Images)
			}
			if tt.wantAudios != nil {
				assert.Equal(t, tt.wantAudios, result.Audios)
			} else {
				assert.Empty(t, result.Audios)
			}
			assert.Equal(t, tt.wantStructured, result.StructuredContent)
		})
	}
}

// callToolResult is a helper to build a CallToolResult from content blocks.
func callToolResult(content ...mcp.Content) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: content}
}
