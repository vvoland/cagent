package messages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripBorderChars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no border chars",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "thick border left",
			input:    "┃ Hello World",
			expected: " Hello World",
		},
		{
			name:     "normal border left",
			input:    "│ Hello World",
			expected: " Hello World",
		},
		{
			name:     "double border left",
			input:    "║ Hello World",
			expected: " Hello World",
		},
		{
			name:     "multiple border chars",
			input:    "┃ Hello ┃ World ┃",
			expected: " Hello  World ",
		},
		{
			name:     "mixed border types",
			input:    "┃│║ Hello World",
			expected: " Hello World",
		},
		{
			name:     "border box corners",
			input:    "┌─────┐\n│Hello│\n└─────┘",
			expected: "\nHello\n",
		},
		{
			name:     "rounded border corners",
			input:    "╭─────╮\n│Hello│\n╰─────╯",
			expected: "\nHello\n",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only border chars",
			input:    "┃│║┌┐└┘",
			expected: "",
		},
		{
			name:     "unicode content preserved",
			input:    "┃ Hello 世界 🎉",
			expected: " Hello 世界 🎉",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := stripBorderChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBoxDrawingCharsMap(t *testing.T) {
	t.Parallel()

	// Ensure all common border characters are in the map
	commonBorderChars := []rune{
		// Thick border
		'┃', '━', '┏', '┓', '┗', '┛',
		// Normal border
		'│', '─', '┌', '┐', '└', '┘',
		// Double border
		'║', '═', '╔', '╗', '╚', '╝',
		// Rounded border
		'╭', '╮', '╯', '╰',
		// T-junctions and crosses (thick)
		'┣', '┫', '┳', '┻', '╋',
		// T-junctions and crosses (normal)
		'├', '┤', '┬', '┴', '┼',
		// T-junctions and crosses (double)
		'╠', '╣', '╦', '╩', '╬',
	}

	for _, char := range commonBorderChars {
		assert.True(t, boxDrawingChars[char], "Character %q (U+%04X) should be in boxDrawingChars map", char, char)
	}
}

func TestDisplayWidthToRuneIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		targetWidth int
		expected    int
	}{
		{
			name:        "empty string",
			input:       "",
			targetWidth: 0,
			expected:    0,
		},
		{
			name:        "zero width",
			input:       "Hello",
			targetWidth: 0,
			expected:    0,
		},
		{
			name:        "negative width",
			input:       "Hello",
			targetWidth: -1,
			expected:    0,
		},
		{
			name:        "ASCII first char",
			input:       "Hello",
			targetWidth: 1,
			expected:    1,
		},
		{
			name:        "ASCII middle char",
			input:       "Hello",
			targetWidth: 3,
			expected:    3,
		},
		{
			name:        "ASCII beyond end",
			input:       "Hello",
			targetWidth: 10,
			expected:    5,
		},
		{
			name:        "wide char (CJK)",
			input:       "世界",
			targetWidth: 2, // First wide char takes 2 columns
			expected:    1, // So width 2 points to index 1
		},
		{
			name:        "mixed ASCII and wide",
			input:       "Hi世界",
			targetWidth: 4, // "Hi" = 2, then first wide char adds 2 more
			expected:    3, // Index 3 is the second wide char
		},
		{
			name:        "bullet point",
			input:       "• item",
			targetWidth: 1,
			expected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := displayWidthToRuneIndex(tt.input, tt.targetWidth)
			assert.Equal(t, tt.expected, result)
		})
	}
}
