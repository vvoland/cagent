package markdown

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stripANSI removes ANSI escape sequences from a string.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func TestFastRendererBasicText(t *testing.T) {
	t.Parallel()

	r := NewFastRenderer(80)
	result, err := r.Render("Hello, world!")
	require.NoError(t, err)
	assert.Contains(t, result, "Hello, world!")
}

func TestFastRendererEmptyInput(t *testing.T) {
	t.Parallel()

	r := NewFastRenderer(80)
	result, err := r.Render("")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestFastRendererHeadings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"h1", "# Heading 1", "Heading 1"},
		{"h2", "## Heading 2", "Heading 2"},
		{"h3", "### Heading 3", "Heading 3"},
		{"h4", "#### Heading 4", "Heading 4"},
		{"h5", "##### Heading 5", "Heading 5"},
		{"h6", "###### Heading 6", "Heading 6"},
		{"h1 with trailing hashes", "# Heading #", "Heading"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewFastRenderer(80)
			result, err := r.Render(tt.input)
			require.NoError(t, err)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestFastRendererCodeBlocks(t *testing.T) {
	t.Parallel()

	input := "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"
	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)

	// Should contain the code content
	plain := stripANSI(result)
	assert.Contains(t, plain, "func")
	assert.Contains(t, plain, "main")
	assert.Contains(t, plain, "Println")
}

func TestFastRendererCodeBlocksNoLanguage(t *testing.T) {
	t.Parallel()

	input := "```\nsome code here\n```"
	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)
	assert.Contains(t, stripANSI(result), "some code here")
}

func TestFastRendererInlineCode(t *testing.T) {
	t.Parallel()

	input := "Use `fmt.Println` to print"
	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)
	assert.Contains(t, result, "fmt.Println")
}

func TestFastRendererBold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"double asterisk", "This is **bold** text", "bold"},
		{"double underscore", "This is __bold__ text", "bold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewFastRenderer(80)
			result, err := r.Render(tt.input)
			require.NoError(t, err)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestFastRendererItalic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"single asterisk", "This is *italic* text", "italic"},
		{"single underscore", "This is _italic_ text", "italic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewFastRenderer(80)
			result, err := r.Render(tt.input)
			require.NoError(t, err)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestFastRendererStrikethrough(t *testing.T) {
	t.Parallel()

	input := "This is ~~strikethrough~~ text"
	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)
	assert.Contains(t, stripANSI(result), "strikethrough")
}

func TestFastRendererLinks(t *testing.T) {
	t.Parallel()

	input := "Check out [this link](https://example.com)"
	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)
	plain := stripANSI(result)
	assert.Contains(t, plain, "this link")
	assert.Contains(t, plain, "example.com")
}

func TestFastRendererUnorderedLists(t *testing.T) {
	t.Parallel()

	input := `- Item 1
- Item 2
- Item 3`

	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)

	assert.Contains(t, result, "Item 1")
	assert.Contains(t, result, "Item 2")
	assert.Contains(t, result, "Item 3")
	assert.Contains(t, result, "•")
}

func TestFastRendererOrderedLists(t *testing.T) {
	t.Parallel()

	input := `1. First
2. Second
3. Third`

	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)

	assert.Contains(t, result, "First")
	assert.Contains(t, result, "Second")
	assert.Contains(t, result, "Third")
}

func TestFastRendererTaskLists(t *testing.T) {
	t.Parallel()

	input := `- [ ] Todo item
- [x] Done item`

	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)

	assert.Contains(t, result, "Todo item")
	assert.Contains(t, result, "Done item")
}

func TestFastRendererBlockquotes(t *testing.T) {
	t.Parallel()

	input := "> This is a quote"
	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)
	assert.Contains(t, result, "This is a quote")
}

func TestFastRendererHorizontalRule(t *testing.T) {
	t.Parallel()

	tests := []string{"---", "***", "___"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			r := NewFastRenderer(80)
			result, err := r.Render(input)
			require.NoError(t, err)
			assert.Contains(t, result, "---")
		})
	}
}

func TestFastRendererTables(t *testing.T) {
	t.Parallel()

	input := `| Name | Age |
|------|-----|
| Alice | 30 |
| Bob | 25 |`

	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)

	plain := stripANSI(result)
	assert.Contains(t, plain, "Name")
	assert.Contains(t, plain, "Age")
	assert.Contains(t, plain, "Alice")
	assert.Contains(t, plain, "Bob")
	assert.Contains(t, plain, "30")
	assert.Contains(t, plain, "25")
}

func TestFastRendererEscapedCharacters(t *testing.T) {
	t.Parallel()

	input := `\*not italic\* and \**not bold\**`
	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)
	assert.Contains(t, result, "*not italic*")
}

func TestFastRendererNestedLists(t *testing.T) {
	t.Parallel()

	input := `- Item 1
  - Nested 1
  - Nested 2
- Item 2`

	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)

	assert.Contains(t, result, "Item 1")
	assert.Contains(t, result, "Nested 1")
	assert.Contains(t, result, "Nested 2")
	assert.Contains(t, result, "Item 2")
}

func TestFastRendererMixedContent(t *testing.T) {
	t.Parallel()

	input := `# Main Title

This is a paragraph with **bold** and *italic* text.

## Code Example

` + "```go\nfmt.Println(\"Hello\")\n```" + `

- List item 1
- List item 2

> A blockquote

---

The end.`

	r := NewFastRenderer(80)
	result, err := r.Render(input)
	require.NoError(t, err)

	plain := stripANSI(result)
	assert.Contains(t, plain, "Main Title")
	assert.Contains(t, plain, "bold")
	assert.Contains(t, plain, "italic")
	assert.Contains(t, plain, "Println")
	assert.Contains(t, plain, "List item")
	assert.Contains(t, plain, "blockquote")
	assert.Contains(t, plain, "The end")
}

func TestFastRendererWordWrapping(t *testing.T) {
	t.Parallel()

	longText := "This is a very long line that should be wrapped when the width is constrained to something reasonable like forty characters."
	r := NewFastRenderer(40)
	result, err := r.Render(longText)
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	// Should wrap into multiple lines
	assert.Greater(t, len(lines), 1)
}

func TestFastRendererRendererInterface(t *testing.T) {
	t.Parallel()

	// Ensure FastRenderer implements Renderer interface
	var r Renderer = NewFastRenderer(80)
	result, err := r.Render("test")
	require.NoError(t, err)
	assert.Contains(t, result, "test")
}

// Benchmark tests to compare performance
var benchmarkInput = `# Performance Test Document

This is a comprehensive markdown document designed to test the performance of the markdown renderer.

## Features

Here's a list of features we're testing:

- **Bold text** for emphasis
- *Italic text* for subtle emphasis
- ` + "`inline code`" + ` for technical terms
- ~~Strikethrough~~ for deleted content
- [Links](https://example.com) for references

## Code Block

` + "```go" + `
package main

import "fmt"

func main() {
    for i := 0; i < 10; i++ {
        fmt.Printf("Hello %d\n", i)
    }
}
` + "```" + `

## Blockquote

> This is an important quote that should be displayed
> with proper formatting and styling.

## Task List

- [x] Implement fast renderer
- [x] Add syntax highlighting
- [ ] Add more tests
- [ ] Optimize further

---

## Conclusion

This document contains various markdown elements that are commonly used in LLM responses.
The fast renderer should handle all of these efficiently.
`

func BenchmarkFastRenderer(b *testing.B) {
	r := NewFastRenderer(80)
	b.ResetTimer()
	for range b.N {
		_, _ = r.Render(benchmarkInput)
	}
}

func BenchmarkGlamourRenderer(b *testing.B) {
	r := NewGlamourRenderer(80)
	b.ResetTimer()
	for range b.N {
		_, _ = r.Render(benchmarkInput)
	}
}

func BenchmarkFastRendererSmall(b *testing.B) {
	r := NewFastRenderer(80)
	input := "Hello **world**, this is a *test*."
	b.ResetTimer()
	for range b.N {
		_, _ = r.Render(input)
	}
}

func BenchmarkGlamourRendererSmall(b *testing.B) {
	r := NewGlamourRenderer(80)
	input := "Hello **world**, this is a *test*."
	b.ResetTimer()
	for range b.N {
		_, _ = r.Render(input)
	}
}

func BenchmarkFastRendererCodeBlock(b *testing.B) {
	r := NewFastRenderer(80)
	input := "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"
	b.ResetTimer()
	for range b.N {
		_, _ = r.Render(input)
	}
}

func BenchmarkGlamourRendererCodeBlock(b *testing.B) {
	r := NewGlamourRenderer(80)
	input := "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"
	b.ResetTimer()
	for range b.N {
		_, _ = r.Render(input)
	}
}
