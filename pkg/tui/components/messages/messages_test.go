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

func TestIsWordChar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    rune
		expected bool
	}{
		// Letters
		{name: "lowercase letter", input: 'a', expected: true},
		{name: "uppercase letter", input: 'Z', expected: true},
		{name: "digit", input: '5', expected: true},
		{name: "underscore", input: '_', expected: true},
		// Non-ASCII word characters
		{name: "unicode letter", input: 'é', expected: true},
		{name: "CJK character", input: '世', expected: true},
		// Non-word characters
		{name: "space", input: ' ', expected: false},
		{name: "period", input: '.', expected: false},
		{name: "comma", input: ',', expected: false},
		{name: "parenthesis", input: '(', expected: false},
		{name: "hyphen", input: '-', expected: false},
		{name: "colon", input: ':', expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isWordChar(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRuneIndexToDisplayWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		runeIdx  int
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			runeIdx:  0,
			expected: 0,
		},
		{
			name:     "ASCII string start",
			input:    "Hello",
			runeIdx:  0,
			expected: 0,
		},
		{
			name:     "ASCII string middle",
			input:    "Hello",
			runeIdx:  3,
			expected: 3,
		},
		{
			name:     "ASCII string end",
			input:    "Hello",
			runeIdx:  5,
			expected: 5,
		},
		{
			name:     "wide char (CJK) first",
			input:    "世界",
			runeIdx:  1,
			expected: 2, // First wide char takes 2 columns
		},
		{
			name:     "wide char (CJK) second",
			input:    "世界",
			runeIdx:  2,
			expected: 4, // Both wide chars take 4 columns total
		},
		{
			name:     "mixed ASCII and wide",
			input:    "Hi世界",
			runeIdx:  3,
			expected: 4, // "Hi" = 2, plus first wide char = 2
		},
		{
			name:     "index beyond string length",
			input:    "Hello",
			runeIdx:  10,
			expected: 5, // Clamped to string length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := runeIndexToDisplayWidth(tt.input, tt.runeIdx)
			assert.Equal(t, tt.expected, result)
		})
	}
}
