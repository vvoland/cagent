package messages

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// urlAtPosition extracts a URL from the rendered line at the given display column.
// Returns the URL string if found, or empty string if the click position is not on a URL.
func urlAtPosition(renderedLine string, col int) string {
	plainLine := ansi.Strip(renderedLine)
	if plainLine == "" {
		return ""
	}

	// Find all URL spans in the plain text
	for _, span := range findURLSpans(plainLine) {
		if col >= span.startCol && col < span.endCol {
			return span.url
		}
	}
	return ""
}

type urlSpan struct {
	url      string
	startCol int // display column where URL starts
	endCol   int // display column where URL ends (exclusive)
}

// findURLSpans finds all URLs in plain text and returns their display column ranges.
func findURLSpans(text string) []urlSpan {
	var spans []urlSpan
	runes := []rune(text)
	n := len(runes)

	for i := 0; i < n; {
		// Look for http:// or https://
		remaining := string(runes[i:])
		var prefixLen int
		switch {
		case strings.HasPrefix(remaining, "https://"):
			prefixLen = len("https://")
		case strings.HasPrefix(remaining, "http://"):
			prefixLen = len("http://")
		default:
			i++
			continue
		}

		// Must not be preceded by a word character (avoid matching mid-word)
		if i > 0 && isURLWordChar(runes[i-1]) {
			i++
			continue
		}

		urlStart := i
		j := i + prefixLen
		// Extend to cover the URL body
		for j < n && isURLChar(runes[j]) {
			j++
		}
		// Strip common trailing punctuation that's unlikely part of the URL
		for j > urlStart+prefixLen && isTrailingPunct(runes[j-1]) {
			j--
		}
		// Balance parentheses: strip trailing ')' only if unmatched
		url := string(runes[urlStart:j])
		url = balanceParens(url)
		j = urlStart + len([]rune(url))

		startCol := runeSliceWidth(runes[:urlStart])
		endCol := startCol + runeSliceWidth(runes[urlStart:j])

		spans = append(spans, urlSpan{
			url:      url,
			startCol: startCol,
			endCol:   endCol,
		})
		i = j
	}
	return spans
}

func runeSliceWidth(runes []rune) int {
	w := 0
	for _, r := range runes {
		w += runewidth.RuneWidth(r)
	}
	return w
}

func isURLChar(r rune) bool {
	if r <= ' ' || r == '"' || r == '<' || r == '>' || r == '{' || r == '}' || r == '|' || r == '\\' || r == '^' || r == '`' {
		return false
	}
	return true
}

func isURLWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isTrailingPunct(r rune) bool {
	return r == '.' || r == ',' || r == ';' || r == ':' || r == '!' || r == '?'
}

// balanceParens strips a trailing ')' if there are more closing than opening parens.
// This handles the common case of URLs wrapped in parentheses like (https://example.com).
func balanceParens(url string) string {
	if !strings.HasSuffix(url, ")") {
		return url
	}
	open := strings.Count(url, "(")
	if strings.Count(url, ")") > open {
		return url[:len(url)-1]
	}
	return url
}

// urlAt returns the URL at the given global line and display column, or empty string.
func (m *model) urlAt(line, col int) string {
	m.ensureAllItemsRendered()
	if line < 0 || line >= len(m.renderedLines) {
		return ""
	}
	return urlAtPosition(m.renderedLines[line], col)
}
