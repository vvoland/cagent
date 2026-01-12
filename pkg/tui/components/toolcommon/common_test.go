package toolcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryFixPartialJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		shouldFix bool
	}{
		{
			name:      "empty string",
			input:     "",
			expected:  "",
			shouldFix: false,
		},
		{
			name:      "not json object",
			input:     "hello",
			expected:  "hello",
			shouldFix: false,
		},
		{
			name:      "just opening brace",
			input:     `{`,
			expected:  `{}`,
			shouldFix: true,
		},
		{
			name:      "partial key",
			input:     `{"path`,
			expected:  `{"path"}`,
			shouldFix: true,
		},
		{
			name:      "key with colon",
			input:     `{"path":`,
			expected:  `{"path":}`,
			shouldFix: true,
		},
		{
			name:      "incomplete string value",
			input:     `{"path": "/tmp/fi`,
			expected:  `{"path": "/tmp/fi"}`,
			shouldFix: true,
		},
		{
			name:      "complete string missing brace",
			input:     `{"path": "/tmp/file"`,
			expected:  `{"path": "/tmp/file"}`,
			shouldFix: true,
		},
		{
			name:      "trailing comma",
			input:     `{"path": "/tmp/file",`,
			expected:  `{"path": "/tmp/file",}`,
			shouldFix: true,
		},
		{
			name:      "nested object incomplete",
			input:     `{"outer": {"inner": "val`,
			expected:  `{"outer": {"inner": "val"}}`,
			shouldFix: true,
		},
		{
			name:      "array incomplete",
			input:     `{"paths": ["/tmp/a", "/tmp/b`,
			expected:  `{"paths": ["/tmp/a", "/tmp/b"]}`,
			shouldFix: true,
		},
		{
			name:      "escaped quote in string",
			input:     `{"msg": "hello \"world`,
			expected:  `{"msg": "hello \"world"}`,
			shouldFix: true,
		},
		{
			name:      "complete json",
			input:     `{"path": "/tmp/file"}`,
			expected:  `{"path": "/tmp/file"}`,
			shouldFix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := tryFixPartialJSON(tt.input)
			assert.Equal(t, tt.shouldFix, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePartialArgs(t *testing.T) {
	type testArgs struct {
		Path string `json:"path"`
		Cmd  string `json:"cmd"`
	}

	tests := []struct {
		name     string
		input    string
		wantPath string
		wantCmd  string
		wantErr  bool
	}{
		{
			name:     "complete JSON",
			input:    `{"path": "/tmp/file", "cmd": "ls -la"}`,
			wantPath: "/tmp/file",
			wantCmd:  "ls -la",
			wantErr:  false,
		},
		{
			name:     "partial JSON - missing closing brace",
			input:    `{"path": "/tmp/file"`,
			wantPath: "/tmp/file",
			wantCmd:  "",
			wantErr:  false,
		},
		{
			name:     "partial JSON - incomplete string value",
			input:    `{"path": "/tmp/fi`,
			wantPath: "/tmp/fi",
			wantCmd:  "",
			wantErr:  false,
		},
		{
			name:     "partial JSON - only key",
			input:    `{"path":`,
			wantPath: "",
			wantCmd:  "",
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			wantPath: "",
			wantCmd:  "",
			wantErr:  true,
		},
		{
			name:     "just opening brace",
			input:    "{",
			wantPath: "",
			wantCmd:  "",
			wantErr:  false,
		},
		{
			name:     "nested object in progress",
			input:    `{"path": "/tmp", "nested": {"key": "val`,
			wantPath: "/tmp",
			wantCmd:  "",
			wantErr:  false,
		},
		{
			name:     "array value in progress",
			input:    `{"path": "/tmp", "items": ["a", "b`,
			wantPath: "/tmp",
			wantCmd:  "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseArgs[testArgs](tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, result.Path)
			assert.Equal(t, tt.wantCmd, result.Cmd)
		})
	}
}

func TestWrapLines(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected []string
	}{
		// Basic wrapping cases
		{
			name:     "simple text within width",
			text:     "hello world",
			width:    20,
			expected: []string{"hello world"},
		},
		{
			name:     "text exactly at width",
			text:     "hello",
			width:    5,
			expected: []string{"hello"},
		},
		{
			name:     "single line longer than width",
			text:     "hello world this is a long line",
			width:    10,
			expected: []string{"hello worl", "d this is ", "a long lin", "e"},
		},
		{
			name:     "text wraps at exact boundary",
			text:     "abcdefghij",
			width:    5,
			expected: []string{"abcde", "fghij"},
		},
		{
			name:     "text wraps with remainder",
			text:     "abcdefghijk",
			width:    5,
			expected: []string{"abcde", "fghij", "k"},
		},

		// Multi-line input cases
		{
			name:     "multiple short lines",
			text:     "line1\nline2\nline3",
			width:    10,
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "mixed short and long lines",
			text:     "short\nthis is a very long line that needs wrapping\nanother",
			width:    10,
			expected: []string{"short", "this is a ", "very long ", "line that ", "needs wrap", "ping", "another"},
		},
		{
			name:     "empty lines preserved",
			text:     "line1\n\nline3",
			width:    10,
			expected: []string{"line1", "", "line3"},
		},
		{
			name:     "lines with trailing newline",
			text:     "line1\nline2\n",
			width:    10,
			expected: []string{"line1", "line2", ""},
		},

		// Edge cases
		{
			name:     "empty string",
			text:     "",
			width:    10,
			expected: []string{""},
		},
		{
			name:     "only newlines",
			text:     "\n\n\n",
			width:    10,
			expected: []string{"", "", "", ""},
		},
		{
			name:     "zero width",
			text:     "hello world",
			width:    0,
			expected: []string{"hello world"},
		},
		{
			name:     "negative width",
			text:     "hello world",
			width:    -5,
			expected: []string{"hello world"},
		},
		{
			name:     "width of 1",
			text:     "hello",
			width:    1,
			expected: []string{"h", "e", "l", "l", "o"},
		},
		{
			name:     "single character",
			text:     "a",
			width:    1,
			expected: []string{"a"},
		},
		{
			name:     "single character with large width",
			text:     "a",
			width:    100,
			expected: []string{"a"},
		},

		// Boundary and special cases
		{
			name:     "text with spaces at boundaries",
			text:     "hello world test",
			width:    6,
			expected: []string{"hello ", "world ", "test"},
		},
		{
			name:     "very long single word",
			text:     "supercalifragilisticexpialidocious",
			width:    10,
			expected: []string{"supercalif", "ragilistic", "expialidoc", "ious"},
		},
		{
			name:     "multiple consecutive newlines",
			text:     "a\n\n\nb",
			width:    5,
			expected: []string{"a", "", "", "b"},
		},
		{
			name:     "line exactly matching width multiple times",
			text:     "12345",
			width:    5,
			expected: []string{"12345"},
		},
		{
			name:     "unicode characters",
			text:     "héllo wörld",
			width:    8,
			expected: []string{"héllo wö", "rld"},
		},
		{
			name:     "tabs and special characters",
			text:     "hello\tworld\ntest",
			width:    8,
			expected: []string{"hello\twor", "ld", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wrapped := WrapLines(tt.text, tt.width)

			assert.Equal(t, tt.expected, wrapped)
		})
	}
}
