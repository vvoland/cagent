package editor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/history"
)

func TestApplySuggestionOverlay(t *testing.T) {
	tests := []struct {
		name         string
		suggestion   string
		textareaText string
		shouldPanic  bool
	}{
		{
			name:         "single line suggestion",
			suggestion:   "complete this",
			textareaText: "hello ",
			shouldPanic:  false,
		},
		{
			name:         "two line suggestion",
			suggestion:   "line 1\nline 2",
			textareaText: "start ",
			shouldPanic:  false,
		},
		{
			name:         "three line suggestion",
			suggestion:   "line 1\nline 2\nline 3",
			textareaText: "start ",
			shouldPanic:  false,
		},
		{
			name:         "many line suggestion",
			suggestion:   "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7",
			textareaText: "start ",
			shouldPanic:  false,
		},
		{
			name:         "empty suggestion",
			suggestion:   "",
			textareaText: "hello",
			shouldPanic:  false,
		},
		{
			name:         "newline only suggestion",
			suggestion:   "\n",
			textareaText: "hello",
			shouldPanic:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a simple editor for testing
			hist, _ := history.New()
			a := &app.App{} // minimal app for testing
			e := New(a, hist).(*editor)

			// Set up textarea with test content
			e.textarea.SetValue(tt.textareaText)
			e.textarea.SetWidth(80)
			e.textarea.SetHeight(10)

			// Set up suggestion
			e.suggestion = tt.suggestion
			e.hasSuggestion = true

			// Create a simple base view
			baseView := e.textarea.View()

			// Test the function
			if tt.shouldPanic {
				assert.Panics(t, func() {
					e.applySuggestionOverlay(baseView)
				})
			} else {
				assert.NotPanics(t, func() {
					result := e.applySuggestionOverlay(baseView)
					// Just verify we get some output
					assert.NotEmpty(t, result)
				})
			}
		})
	}
}

func TestApplySuggestionOverlayScrolledView(t *testing.T) {
	t.Parallel()

	hist, _ := history.New()
	a := &app.App{}
	e := New(a, hist).(*editor)

	// Set up a multi-line textarea that's scrolled
	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\ncurrent"
	e.textarea.SetValue(content)
	e.textarea.SetWidth(80)
	e.textarea.SetHeight(3) // Small height to force scrolling

	// Move cursor to end to scroll to bottom
	e.textarea.MoveToEnd()

	// Set suggestion
	e.suggestion = "suggestion text"
	e.hasSuggestion = true

	baseView := e.textarea.View()

	// Should not panic even with scrolled content
	require.NotPanics(t, func() {
		result := e.applySuggestionOverlay(baseView)
		assert.NotEmpty(t, result)
	})
}

// TestApplySuggestionOverlayWithVariousLineCount tests suggestion overlay positioning
// with 0, 1, 2, 3, and 4 lines of text in the editor.
func TestApplySuggestionOverlayWithVariousLineCount(t *testing.T) {
	tests := []struct {
		name            string
		textareaText    string
		suggestion      string
		expectedLine    int    // Which visual line should contain the suggestion
		expectAfterText string // Text that should appear BEFORE the suggestion on that line (empty if at start)
	}{
		{
			name:            "0 lines - empty editor",
			textareaText:    "",
			suggestion:      "SUGG",
			expectedLine:    0,
			expectAfterText: "", // Suggestion at column 0 (value is empty)
		},
		{
			name:            "1 line - single char",
			textareaText:    "a",
			suggestion:      "SUGG",
			expectedLine:    0,
			expectAfterText: "a",
		},
		{
			name:            "1 line - with text",
			textareaText:    "hello world",
			suggestion:      "SUGG",
			expectedLine:    0,
			expectAfterText: "hello world",
		},
		{
			name:            "1 line ending with newline - cursor on empty second line",
			textareaText:    "hello\n",
			suggestion:      "SUGG",
			expectedLine:    1,  // Cursor is on line 1 (empty line after "hello")
			expectAfterText: "", // Suggestion at column 0 on empty line
		},
		{
			name:            "2 lines",
			textareaText:    "line1\nline2",
			suggestion:      "SUGG",
			expectedLine:    1,
			expectAfterText: "line2",
		},
		{
			name:            "2 lines ending with newline",
			textareaText:    "line1\nline2\n",
			suggestion:      "SUGG",
			expectedLine:    2,  // Cursor is on line 2 (empty line after "line2")
			expectAfterText: "", // Suggestion at column 0 on empty line
		},
		{
			name:            "3 lines",
			textareaText:    "a\nb\nc",
			suggestion:      "SUGG",
			expectedLine:    2,
			expectAfterText: "c",
		},
		{
			name:            "4 lines",
			textareaText:    "a\nb\nc\nd",
			suggestion:      "SUGG",
			expectedLine:    3,
			expectAfterText: "d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hist, _ := history.New()
			a := &app.App{}
			e := New(a, hist).(*editor)

			e.textarea.SetValue(tt.textareaText)
			e.textarea.SetWidth(80)
			e.textarea.SetHeight(10)
			e.textarea.MoveToEnd()

			e.suggestion = tt.suggestion
			e.hasSuggestion = true

			baseView := e.textarea.View()
			result := e.applySuggestionOverlay(baseView)

			// Parse the result and verify suggestion position
			lines := strings.Split(result, "\n")
			require.Greater(t, len(lines), tt.expectedLine,
				"result should have at least %d lines", tt.expectedLine+1)

			// Get the line where we expect the suggestion
			targetLine := stripANSI(lines[tt.expectedLine])

			// The suggestion should appear on this line
			assert.Contains(t, targetLine, tt.suggestion,
				"suggestion %q should appear on line %d", tt.suggestion, tt.expectedLine)

			// Verify the suggestion appears after the expected text
			if tt.expectAfterText != "" {
				expectedContent := tt.expectAfterText + tt.suggestion
				assert.Contains(t, targetLine, expectedContent,
					"line should contain %q, got %q", expectedContent, targetLine)
			}
		})
	}
}

// TestApplySuggestionOverlayWithMultiLineSuggestion tests that multi-line suggestions
// are rendered correctly starting from the target line.
func TestApplySuggestionOverlayWithMultiLineSuggestion(t *testing.T) {
	tests := []struct {
		name           string
		textareaText   string
		suggestion     string
		expectedLine   int // First line of suggestion
		editorHeight   int
		checkFirstLine bool // Whether to verify the first line of suggestion
	}{
		{
			name:           "2-line suggestion on single line text",
			textareaText:   "hello",
			suggestion:     " world\nmore text",
			expectedLine:   0,
			editorHeight:   10,
			checkFirstLine: true,
		},
		{
			name:           "3-line suggestion on 2 line text",
			textareaText:   "line1\nline2",
			suggestion:     " end\nline3\nline4",
			expectedLine:   1,
			editorHeight:   10,
			checkFirstLine: true,
		},
		{
			name:           "multi-line suggestion with scrolled view",
			textareaText:   "a\nb\nc\nd\ne",
			suggestion:     "f\ng\nh",
			expectedLine:   2, // Only showing last 3 lines due to height=3
			editorHeight:   3,
			checkFirstLine: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hist, _ := history.New()
			a := &app.App{}
			e := New(a, hist).(*editor)

			e.textarea.SetValue(tt.textareaText)
			e.textarea.SetWidth(80)
			e.textarea.SetHeight(tt.editorHeight)
			e.textarea.MoveToEnd()

			e.suggestion = tt.suggestion
			e.hasSuggestion = true

			baseView := e.textarea.View()
			result := e.applySuggestionOverlay(baseView)

			// The function should not panic
			require.NotEmpty(t, result)

			// Verify at least the first line of suggestion is present
			if tt.checkFirstLine {
				lines := strings.Split(result, "\n")
				require.Greater(t, len(lines), tt.expectedLine)

				firstSuggestionPart := strings.Split(tt.suggestion, "\n")[0]
				targetLine := stripANSI(lines[tt.expectedLine])
				assert.Contains(t, targetLine, firstSuggestionPart,
					"first part of suggestion %q should appear on line %d, got %q",
					firstSuggestionPart, tt.expectedLine, targetLine)
			}
		})
	}
}

// TestExtractLineText tests the extractLineText helper function.
func TestExtractLineText(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		prompt   string
		expected string
	}{
		{
			name:     "plain text no prompt",
			line:     "hello world       ",
			prompt:   "",
			expected: "hello world",
		},
		{
			name:     "with ANSI codes",
			line:     "\x1b[31mred text\x1b[0m       ",
			prompt:   "",
			expected: "red text",
		},
		{
			name:     "with prompt",
			line:     "> hello       ",
			prompt:   "> ",
			expected: "hello",
		},
		{
			name:     "empty line",
			line:     "          ",
			prompt:   "",
			expected: "",
		},
		{
			name:     "only prompt",
			line:     "> ",
			prompt:   "> ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractLineText(tt.line, tt.prompt)
			assert.Equal(t, tt.expected, result)
		})
	}
}
