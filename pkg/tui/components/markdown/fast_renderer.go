// Package markdown provides a high-performance markdown renderer for terminal output.
// This is a custom implementation optimized for speed, replacing glamour for TUI rendering.
package markdown

import (
	"bytes"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/glamour/v2/ansi"
	xansi "github.com/charmbracelet/x/ansi"
	runewidth "github.com/mattn/go-runewidth"

	"github.com/docker/cagent/pkg/tui/styles"
)

// ansiStyle holds pre-computed ANSI escape sequences for fast rendering.
// This avoids the overhead of lipgloss.Style.Render() which copies large structs.
type ansiStyle struct {
	prefix string // ANSI codes to start the style
	suffix string // ANSI codes to end the style (reset)
}

func (s ansiStyle) render(text string) string {
	if s.prefix == "" {
		return text
	}
	return s.prefix + text + s.suffix
}

// buildAnsiStyle extracts ANSI codes from a lipgloss style by rendering an empty marker.
func buildAnsiStyle(style lipgloss.Style) ansiStyle {
	// Render a marker to extract the ANSI prefix/suffix
	const marker = "\x00"
	rendered := style.Render(marker)
	idx := strings.Index(rendered, marker)
	if idx == -1 {
		return ansiStyle{}
	}
	return ansiStyle{
		prefix: rendered[:idx],
		suffix: rendered[idx+1:],
	}
}

// cachedStyles holds pre-computed styles to avoid repeated MarkdownStyle() calls.
type cachedStyles struct {
	// lipgloss styles (for complex rendering like headings)
	headingStyles   [6]lipgloss.Style
	headingPrefixes [6]string
	styleHR         lipgloss.Style
	styleBlockquote lipgloss.Style
	styleCodeBg     lipgloss.Style

	// ANSI styles (for fast inline rendering)
	ansiBold     ansiStyle
	ansiItalic   ansiStyle
	ansiBoldItal ansiStyle
	ansiStrike   ansiStyle
	ansiCode     ansiStyle
	ansiLink     ansiStyle
	ansiLinkText ansiStyle

	styleTaskTicked  string
	styleTaskUntick  string
	listIndent       int
	blockquoteIndent int
	chromaStyle      *chroma.Style
}

var (
	globalStyles     *cachedStyles
	globalStylesOnce sync.Once
)

func getGlobalStyles() *cachedStyles {
	globalStylesOnce.Do(func() {
		mdStyle := styles.MarkdownStyle()

		styleBold := buildStylePrimitive(mdStyle.Strong)
		styleItalic := buildStylePrimitive(mdStyle.Emph)

		globalStyles = &cachedStyles{
			headingStyles: [6]lipgloss.Style{
				buildStylePrimitive(mdStyle.H1.StylePrimitive),
				buildStylePrimitive(mdStyle.H2.StylePrimitive),
				buildStylePrimitive(mdStyle.H3.StylePrimitive),
				buildStylePrimitive(mdStyle.H4.StylePrimitive),
				buildStylePrimitive(mdStyle.H5.StylePrimitive),
				buildStylePrimitive(mdStyle.H6.StylePrimitive),
			},
			headingPrefixes:  [6]string{"## ", "## ", "### ", "#### ", "##### ", "###### "},
			styleBlockquote:  buildStylePrimitive(mdStyle.BlockQuote.StylePrimitive),
			styleHR:          buildStylePrimitive(mdStyle.HorizontalRule),
			styleCodeBg:      lipgloss.NewStyle(),
			ansiBold:         buildAnsiStyle(styleBold),
			ansiItalic:       buildAnsiStyle(styleItalic),
			ansiBoldItal:     buildAnsiStyle(styleBold.Inherit(styleItalic)),
			ansiStrike:       buildAnsiStyle(buildStylePrimitive(mdStyle.Strikethrough)),
			ansiCode:         buildAnsiStyle(buildStylePrimitive(mdStyle.Code.StylePrimitive)),
			ansiLink:         buildAnsiStyle(buildStylePrimitive(mdStyle.Link)),
			ansiLinkText:     buildAnsiStyle(buildStylePrimitive(mdStyle.LinkText)),
			styleTaskTicked:  mdStyle.Task.Ticked,
			styleTaskUntick:  mdStyle.Task.Unticked,
			listIndent:       int(mdStyle.List.LevelIndent),
			blockquoteIndent: 1,
			chromaStyle:      styles.ChromaStyle(),
		}
		if mdStyle.BlockQuote.Indent != nil {
			globalStyles.blockquoteIndent = int(*mdStyle.BlockQuote.Indent)
		}
		if mdStyle.CodeBlock.BackgroundColor != nil {
			globalStyles.styleCodeBg = globalStyles.styleCodeBg.Background(lipgloss.Color(*mdStyle.CodeBlock.BackgroundColor))
		}
	})
	return globalStyles
}

// FastRenderer is a high-performance markdown renderer optimized for terminal output.
// It directly parses and renders markdown without building an intermediate AST.
type FastRenderer struct {
	width int
}

// NewFastRenderer creates a new fast markdown renderer with the given width.
func NewFastRenderer(width int) *FastRenderer {
	return &FastRenderer{width: width}
}

var parserPool = sync.Pool{
	New: func() any {
		return &parser{
			out: strings.Builder{},
		}
	},
}

// Render parses and renders markdown content to styled terminal output.
func (r *FastRenderer) Render(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	p := parserPool.Get().(*parser)
	p.reset(input, r.width)
	result := p.parse()
	parserPool.Put(p)
	return result, nil
}

// parser holds the state for parsing markdown.
type parser struct {
	input   string
	width   int
	styles  *cachedStyles
	out     strings.Builder
	lines   []string
	lineIdx int
}

func (p *parser) reset(input string, width int) {
	p.input = input
	p.width = width
	p.styles = getGlobalStyles()
	p.lines = strings.Split(input, "\n")
	p.lineIdx = 0
	p.out.Reset()
	p.out.Grow(len(input) * 2) // Pre-allocate for styled output
}

func (p *parser) parse() string {
	for p.lineIdx < len(p.lines) {
		line := p.lines[p.lineIdx]

		switch {
		case p.tryCodeBlock(line):
			// handled inside
		case p.tryHeading(line):
			// handled inside
		case p.tryHorizontalRule(line):
			// handled inside
		case p.tryBlockquote(line):
			// handled inside
		case p.tryTable(line):
			// handled inside
		case p.tryList(line):
			// handled inside
		default:
			// Regular paragraph
			p.renderParagraph()
		}
	}

	return strings.TrimRight(p.out.String(), "\n")
}

// tryCodeBlock checks for fenced code blocks (``` or ~~~)
func (p *parser) tryCodeBlock(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, "~~~") {
		return false
	}

	fence := trimmed[:3]
	lang := strings.TrimSpace(trimmed[3:])
	p.lineIdx++

	var codeLines []string
	for p.lineIdx < len(p.lines) {
		codeLine := p.lines[p.lineIdx]
		if strings.HasPrefix(strings.TrimSpace(codeLine), fence) {
			p.lineIdx++
			break
		}
		codeLines = append(codeLines, codeLine)
		p.lineIdx++
	}

	code := strings.Join(codeLines, "\n")
	p.renderCodeBlock(code, lang)
	return true
}

// tryHeading checks for ATX-style headings (# through ######)
func (p *parser) tryHeading(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "#") {
		return false
	}

	// Count heading level
	level := 0
	for i := 0; i < len(trimmed) && trimmed[i] == '#'; i++ {
		level++
	}
	if level > 6 || level == 0 {
		return false
	}

	// Must have space after #s or be empty
	rest := trimmed[level:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return false
	}

	content := strings.TrimSpace(rest)
	// Remove trailing #s
	content = strings.TrimRight(content, "# \t")

	style := p.headingStyle(level)
	prefix := p.headingPrefix(level)

	rendered := p.renderInline(content)
	p.out.WriteString(style.Render(prefix+rendered) + "\n\n")
	p.lineIdx++
	return true
}

func (p *parser) headingStyle(level int) lipgloss.Style {
	if level >= 1 && level <= 6 {
		return p.styles.headingStyles[level-1]
	}
	return p.styles.headingStyles[0]
}

func (p *parser) headingPrefix(level int) string {
	if level >= 1 && level <= 6 {
		return p.styles.headingPrefixes[level-1]
	}
	return ""
}

// tryHorizontalRule checks for horizontal rules (---, ***, ___)
func (p *parser) tryHorizontalRule(line string) bool {
	if !isHorizontalRule(line) {
		return false
	}
	p.out.WriteString(p.styles.styleHR.Render("\n--------\n") + "\n")
	p.lineIdx++
	return true
}

// tryBlockquote checks for blockquotes (>)
func (p *parser) tryBlockquote(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, ">") {
		return false
	}

	var quoteLines []string
	for p.lineIdx < len(p.lines) {
		l := strings.TrimLeft(p.lines[p.lineIdx], " \t")
		if !strings.HasPrefix(l, ">") {
			break
		}
		// Remove the > and optional space
		content := strings.TrimPrefix(l, ">")
		content = strings.TrimPrefix(content, " ")
		quoteLines = append(quoteLines, content)
		p.lineIdx++
	}

	// Render blockquote content with indent
	indent := strings.Repeat(" ", p.styles.blockquoteIndent)
	for _, ql := range quoteLines {
		rendered := p.renderInline(ql)
		wrapped := p.wrapText(rendered, p.width-p.styles.blockquoteIndent)
		for _, wl := range strings.Split(wrapped, "\n") {
			p.out.WriteString(indent + p.styles.styleBlockquote.Render(wl) + "\n")
		}
	}
	p.out.WriteString("\n")
	return true
}

// tryTable checks for markdown tables
func (p *parser) tryTable(line string) bool {
	// Tables start with | or have | in them
	if !strings.Contains(line, "|") {
		return false
	}

	// Collect all table lines
	var tableLines []string
	for p.lineIdx < len(p.lines) {
		l := p.lines[p.lineIdx]
		if !strings.Contains(l, "|") {
			break
		}
		tableLines = append(tableLines, l)
		p.lineIdx++
	}

	if len(tableLines) < 2 {
		// Need at least header and separator
		// Undo and let paragraph handle it
		p.lineIdx -= len(tableLines)
		return false
	}

	// Check if second line is a separator (contains only -, |, :, and spaces)
	separator := tableLines[1]
	isSeparator := true
	for _, c := range separator {
		if c != '-' && c != '|' && c != ':' && c != ' ' && c != '\t' {
			isSeparator = false
			break
		}
	}

	if !isSeparator {
		// Not a valid table
		p.lineIdx -= len(tableLines)
		return false
	}

	// Parse table cells
	var rows [][]string
	for i, line := range tableLines {
		if i == 1 {
			// Skip separator
			continue
		}
		cells := p.parseTableRow(line)
		rows = append(rows, cells)
	}

	if len(rows) == 0 {
		return false
	}

	// Calculate column widths
	colWidths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				cellWidth := xansi.StringWidth(p.renderInline(cell))
				if cellWidth > colWidths[i] {
					colWidths[i] = cellWidth
				}
			}
		}
	}

	// Render table
	for rowIdx, row := range rows {
		var lineBuilder strings.Builder
		for i, cell := range row {
			if i >= len(colWidths) {
				break
			}
			rendered := p.renderInline(cell)
			padding := colWidths[i] - xansi.StringWidth(rendered)
			if padding < 0 {
				padding = 0
			}
			if rowIdx == 0 {
				// Header row - bold
				lineBuilder.WriteString(p.styles.ansiBold.render(rendered))
			} else {
				lineBuilder.WriteString(rendered)
			}
			lineBuilder.WriteString(strings.Repeat(" ", padding))
			if i < len(row)-1 {
				lineBuilder.WriteString(" │ ")
			}
		}
		p.out.WriteString(lineBuilder.String() + "\n")

		// Add separator after header
		if rowIdx == 0 {
			var sepBuilder strings.Builder
			for i, w := range colWidths {
				sepBuilder.WriteString(strings.Repeat("─", w))
				if i < len(colWidths)-1 {
					sepBuilder.WriteString("─┼─")
				}
			}
			p.out.WriteString(sepBuilder.String() + "\n")
		}
	}

	p.out.WriteString("\n")
	return true
}

func (p *parser) parseTableRow(line string) []string {
	// Remove leading/trailing pipes and split
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")

	var cells []string
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

type listItem struct {
	content string
	ordered bool
	task    bool
	checked bool
}

// parseListItem attempts to parse a list item from a trimmed line.
// Returns the parsed item and true if successful, or zero value and false otherwise.
func parseListItem(line string) (listItem, bool) {
	if len(line) < 2 {
		return listItem{}, false
	}

	// Check unordered list (-, *, +)
	if (line[0] == '-' || line[0] == '*' || line[0] == '+') && line[1] == ' ' {
		content := line[2:]
		item := listItem{content: content}
		if strings.HasPrefix(content, "[ ] ") {
			item.task = true
			item.content = content[4:]
		} else if strings.HasPrefix(content, "[x] ") || strings.HasPrefix(content, "[X] ") {
			item.task = true
			item.checked = true
			item.content = content[4:]
		}
		return item, true
	}

	// Check ordered list (1., 2., etc.)
	dotIdx := strings.Index(line, ".")
	if dotIdx > 0 && dotIdx < 10 && len(line) > dotIdx+1 && line[dotIdx+1] == ' ' {
		for i := range dotIdx {
			if !unicode.IsDigit(rune(line[i])) {
				return listItem{}, false
			}
		}
		return listItem{content: line[dotIdx+2:], ordered: true}, true
	}

	return listItem{}, false
}

// isListStart checks if a line starts a list item
func isListStart(line string) bool {
	_, ok := parseListItem(line)
	return ok
}

// tryList checks for unordered lists (-, *, +) or ordered lists (1., 2., etc.)
func (p *parser) tryList(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if _, ok := parseListItem(trimmed); !ok {
		return false
	}

	for p.lineIdx < len(p.lines) {
		l := p.lines[p.lineIdx]
		ltrimmed := strings.TrimLeft(l, " \t")
		lindent := len(l) - len(ltrimmed)

		item, isListItem := parseListItem(ltrimmed)

		// Empty line handling
		if !isListItem && strings.TrimSpace(l) == "" {
			if p.lineIdx+1 < len(p.lines) {
				nextLine := p.lines[p.lineIdx+1]
				nextTrimmed := strings.TrimLeft(nextLine, " \t")
				if !isListStart(nextTrimmed) {
					break
				}
			} else {
				break
			}
			p.lineIdx++
			continue
		}

		if !isListItem {
			break
		}

		level := lindent / p.styles.listIndent
		bulletIndent := strings.Repeat(" ", level*p.styles.listIndent)

		var bullet string
		switch {
		case item.task && item.checked:
			bullet = p.styles.styleTaskTicked
		case item.task:
			bullet = p.styles.styleTaskUntick
		case item.ordered:
			bullet = ". "
		default:
			bullet = "• "
		}

		rendered := p.renderInline(item.content)
		p.out.WriteString(bulletIndent + bullet + rendered + "\n")
		p.lineIdx++
	}

	p.out.WriteString("\n")
	return true
}

// renderParagraph collects consecutive non-empty lines and renders them as a paragraph.
func (p *parser) renderParagraph() {
	var paraLines []string
	for p.lineIdx < len(p.lines) {
		line := p.lines[p.lineIdx]
		if strings.TrimSpace(line) == "" {
			p.lineIdx++
			break
		}
		// Check if next line starts a block element
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "```") ||
			strings.HasPrefix(trimmed, "~~~") ||
			strings.HasPrefix(trimmed, ">") ||
			isListStart(trimmed) ||
			isHorizontalRule(trimmed) {
			break
		}
		paraLines = append(paraLines, line)
		p.lineIdx++
	}

	if len(paraLines) == 0 {
		return
	}

	// Join lines and render inline elements
	text := strings.Join(paraLines, " ")
	rendered := p.renderInline(text)
	wrapped := p.wrapText(rendered, p.width)
	p.out.WriteString(wrapped + "\n\n")
}

func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	char := trimmed[0]
	if char != '-' && char != '*' && char != '_' {
		return false
	}
	count := 0
	for _, c := range trimmed {
		if c == rune(char) {
			count++
		} else if !unicode.IsSpace(c) {
			return false
		}
	}
	return count >= 3
}

// renderInline processes inline markdown elements: bold, italic, code, links, etc.
func (p *parser) renderInline(text string) string {
	if text == "" {
		return ""
	}

	var out strings.Builder
	out.Grow(len(text) + 64) // Pre-allocate with extra space for ANSI codes
	i := 0
	n := len(text)

	for i < n {
		// Check for escaped characters
		if text[i] == '\\' && i+1 < n {
			out.WriteByte(text[i+1])
			i += 2
			continue
		}

		// Check for inline code
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				code := text[i+1 : i+1+end]
				out.WriteString(p.styles.ansiCode.render(code))
				i = i + 1 + end + 1
				continue
			}
		}

		// Check for bold (**text** or __text__)
		if i+1 < n && ((text[i] == '*' && text[i+1] == '*') || (text[i] == '_' && text[i+1] == '_')) {
			delim := text[i : i+2]
			end := strings.Index(text[i+2:], delim)
			if end != -1 {
				inner := text[i+2 : i+2+end]
				// Check for bold+italic (***text***)
				if strings.HasPrefix(inner, "*") && strings.HasSuffix(inner, "*") && len(inner) >= 2 {
					innerText := inner[1 : len(inner)-1]
					out.WriteString(p.styles.ansiBoldItal.render(p.renderInline(innerText)))
				} else {
					out.WriteString(p.styles.ansiBold.render(p.renderInline(inner)))
				}
				i = i + 2 + end + 2
				continue
			}
		}

		// Check for italic (*text* or _text_) - but not in the middle of words for _
		if text[i] == '*' || (text[i] == '_' && (i == 0 || !isWord(text[i-1]))) {
			delim := text[i]
			end := -1
			for j := i + 1; j < n; j++ {
				if text[j] == delim {
					// For underscore, check it's not in the middle of a word
					if delim == '_' && j+1 < n && isWord(text[j+1]) {
						continue
					}
					end = j
					break
				}
			}
			if end != -1 && end > i+1 {
				inner := text[i+1 : end]
				out.WriteString(p.styles.ansiItalic.render(p.renderInline(inner)))
				i = end + 1
				continue
			}
		}

		// Check for strikethrough (~~text~~)
		if i+1 < n && text[i] == '~' && text[i+1] == '~' {
			end := strings.Index(text[i+2:], "~~")
			if end != -1 {
				inner := text[i+2 : i+2+end]
				out.WriteString(p.styles.ansiStrike.render(p.renderInline(inner)))
				i = i + 2 + end + 2
				continue
			}
		}

		// Check for links [text](url)
		switch c := text[i]; c {
		case '[':
			closeBracket := findClosingBracket(text[i:])
			if closeBracket != -1 && i+closeBracket+1 < n && text[i+closeBracket+1] == '(' {
				linkText := text[i+1 : i+closeBracket]
				rest := text[i+closeBracket+2:]
				closeParen := strings.Index(rest, ")")
				if closeParen != -1 {
					url := rest[:closeParen]
					styledText := p.styles.ansiLinkText.render(linkText)
					if linkText != url {
						out.WriteString(styledText + " " + p.styles.ansiLink.render("("+url+")"))
					} else {
						out.WriteString(p.styles.ansiLink.render(linkText))
					}
					i = i + closeBracket + 2 + closeParen + 1
					continue
				}
			}
			fallthrough
		default:
			// Regular character
			out.WriteByte(text[i])
			i++
		}
	}

	return out.String()
}

func findClosingBracket(text string) int {
	depth := 0
	for i, c := range text {
		switch c {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isWord(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// renderCodeBlock renders a fenced code block with syntax highlighting.
func (p *parser) renderCodeBlock(code, lang string) {
	if code == "" {
		p.out.WriteString("\n")
		return
	}

	// Get syntax highlighting tokens
	tokens := p.syntaxHighlight(code, lang)

	// Calculate content width with margin
	const margin = 2
	contentWidth := p.width - margin*2
	if contentWidth < 20 {
		contentWidth = p.width
	}

	indent := "  " // Pre-computed 2-space indent

	// Pre-compute background padding style
	bgStyle := buildAnsiStyle(p.styles.styleCodeBg)

	// Process tokens line by line for better performance
	var lineBuilder strings.Builder
	lineBuilder.Grow(contentWidth + 32)
	lineWidth := 0

	flushLine := func() {
		lineContent := lineBuilder.String()
		// Pad to full width
		padWidth := contentWidth - lineWidth
		if padWidth > 0 {
			lineContent += bgStyle.render(strings.Repeat(" ", padWidth))
		}
		p.out.WriteString(indent)
		p.out.WriteString(lineContent)
		p.out.WriteByte('\n')
		lineBuilder.Reset()
		lineWidth = 0
	}

	for _, tok := range tokens {
		text := tok.text

		// Process text, splitting by newlines and handling tabs
		start := 0
		for i := range len(text) {
			if text[i] == '\n' {
				// Render text before newline
				if i > start {
					segment := text[start:i]
					segment = expandTabs(segment, lineWidth)
					lineBuilder.WriteString(tok.style.render(segment))
					lineWidth += stringDisplayWidth(segment)
				}
				flushLine()
				start = i + 1
			}
		}
		// Render remaining text
		if start < len(text) {
			segment := text[start:]
			segment = expandTabs(segment, lineWidth)
			lineBuilder.WriteString(tok.style.render(segment))
			lineWidth += stringDisplayWidth(segment)
		}
	}

	// Flush remaining content
	if lineBuilder.Len() > 0 {
		flushLine()
	}

	p.out.WriteByte('\n')
}

// expandTabs replaces tabs with spaces based on current position
func expandTabs(s string, currentWidth int) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	var result strings.Builder
	width := currentWidth
	for _, r := range s {
		if r == '\t' {
			spaces := 4 - (width % 4)
			result.WriteString(strings.Repeat(" ", spaces))
			width += spaces
		} else {
			result.WriteRune(r)
			width += runewidth.RuneWidth(r)
		}
	}
	return result.String()
}

// stringDisplayWidth calculates the display width of a string (handling unicode)
func stringDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		width += runewidth.RuneWidth(r)
	}
	return width
}

type token struct {
	text  string
	style ansiStyle
}

var (
	lexerCache   = make(map[string]chroma.Lexer)
	lexerCacheMu sync.RWMutex

	// Cache for chroma token type to ansiStyle conversion (with code bg)
	chromaStyleCache   = make(map[chroma.TokenType]ansiStyle)
	chromaStyleCacheMu sync.RWMutex
)

func (p *parser) syntaxHighlight(code, lang string) []token {
	var lexer chroma.Lexer

	if lang != "" {
		// Try cache first
		lexerCacheMu.RLock()
		lexer = lexerCache[lang]
		lexerCacheMu.RUnlock()

		if lexer == nil {
			lexer = lexers.Get(lang)
			if lexer == nil {
				// Try with file extension
				lexer = lexers.Match("file." + lang)
			}
			if lexer != nil {
				lexer = chroma.Coalesce(lexer)
				lexerCacheMu.Lock()
				lexerCache[lang] = lexer
				lexerCacheMu.Unlock()
			}
		}
	}

	if lexer == nil {
		// No highlighting - return plain text with code background
		return []token{{text: code, style: p.getCodeStyle(chroma.None)}}
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return []token{{text: code, style: p.getCodeStyle(chroma.None)}}
	}

	chromaTokens := iterator.Tokens()
	tokens := make([]token, 0, len(chromaTokens))
	for _, tok := range chromaTokens {
		if tok.Value == "" {
			continue
		}
		tokens = append(tokens, token{
			text:  tok.Value,
			style: p.getCodeStyle(tok.Type),
		})
	}

	return tokens
}

func (p *parser) getCodeStyle(tokenType chroma.TokenType) ansiStyle {
	chromaStyleCacheMu.RLock()
	style, ok := chromaStyleCache[tokenType]
	chromaStyleCacheMu.RUnlock()
	if ok {
		return style
	}

	// Build lipgloss style with code background inherited
	lipStyle := chromaToLipgloss(tokenType, p.styles.chromaStyle).Inherit(p.styles.styleCodeBg)
	style = buildAnsiStyle(lipStyle)

	chromaStyleCacheMu.Lock()
	chromaStyleCache[tokenType] = style
	chromaStyleCacheMu.Unlock()
	return style
}

func chromaToLipgloss(tokenType chroma.TokenType, style *chroma.Style) lipgloss.Style {
	entry := style.Get(tokenType)
	lipStyle := lipgloss.NewStyle()

	if entry.Colour.IsSet() {
		lipStyle = lipStyle.Foreground(lipgloss.Color(entry.Colour.String()))
	}
	if entry.Bold == chroma.Yes {
		lipStyle = lipStyle.Bold(true)
	}
	if entry.Italic == chroma.Yes {
		lipStyle = lipStyle.Italic(true)
	}
	if entry.Underline == chroma.Yes {
		lipStyle = lipStyle.Underline(true)
	}

	return lipStyle
}

// wrapText wraps text to the given width, respecting ANSI escape sequences.
func (p *parser) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	var currentLine bytes.Buffer
	currentWidth := 0

	// Split into words
	words := splitWords(text)

	for i, word := range words {
		wordWidth := xansi.StringWidth(word)

		// If word alone exceeds width, break it
		if wordWidth > width {
			// Flush current line first
			if currentLine.Len() > 0 {
				result.WriteString(currentLine.String())
				result.WriteByte('\n')
				currentLine.Reset()
			}
			// Break the long word
			broken := breakWord(word, width)
			for j, part := range broken {
				if j > 0 {
					result.WriteByte('\n')
				}
				result.WriteString(part)
			}
			currentWidth = xansi.StringWidth(broken[len(broken)-1])
			if len(broken) > 1 {
				result.WriteByte('\n')
				currentWidth = 0
			}
			continue
		}

		// Check if word fits on current line
		spaceWidth := 0
		if currentWidth > 0 {
			spaceWidth = 1
		}

		if currentWidth+spaceWidth+wordWidth > width {
			// Start new line
			result.WriteString(currentLine.String())
			result.WriteByte('\n')
			currentLine.Reset()
			currentWidth = 0
			spaceWidth = 0
		}

		// Add space if not at start of line
		if spaceWidth > 0 {
			currentLine.WriteByte(' ')
			currentWidth++
		}

		currentLine.WriteString(word)
		currentWidth += wordWidth

		_ = i // silence unused warning
	}

	// Flush remaining content
	if currentLine.Len() > 0 {
		result.WriteString(currentLine.String())
	}

	return result.String()
}

func splitWords(text string) []string {
	var words []string
	var current strings.Builder
	inAnsi := false

	for i := 0; i < len(text); {
		if text[i] == '\x1b' {
			// Start of ANSI sequence
			inAnsi = true
			current.WriteByte(text[i])
			i++
			continue
		}
		if inAnsi {
			current.WriteByte(text[i])
			if (text[i] >= 'a' && text[i] <= 'z') || (text[i] >= 'A' && text[i] <= 'Z') {
				inAnsi = false
			}
			i++
			continue
		}

		if text[i] == ' ' || text[i] == '\t' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			i++
			continue
		}

		r, size := utf8.DecodeRuneInString(text[i:])
		current.WriteRune(r)
		i += size
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func breakWord(word string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{word}
	}

	var parts []string
	var current strings.Builder
	currentWidth := 0
	inAnsi := false
	var ansiSeq strings.Builder

	for i := 0; i < len(word); {
		if word[i] == '\x1b' {
			inAnsi = true
			ansiSeq.WriteByte(word[i])
			i++
			continue
		}
		if inAnsi {
			ansiSeq.WriteByte(word[i])
			if (word[i] >= 'a' && word[i] <= 'z') || (word[i] >= 'A' && word[i] <= 'Z') {
				inAnsi = false
				current.WriteString(ansiSeq.String())
				ansiSeq.Reset()
			}
			i++
			continue
		}

		r, size := utf8.DecodeRuneInString(word[i:])
		rw := runewidth.RuneWidth(r)

		if currentWidth+rw > maxWidth && currentWidth > 0 {
			parts = append(parts, current.String())
			current.Reset()
			currentWidth = 0
		}

		current.WriteRune(r)
		currentWidth += rw
		i += size
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// buildStylePrimitive converts an ansi.StylePrimitive to a lipgloss.Style.
func buildStylePrimitive(sp ansi.StylePrimitive) lipgloss.Style {
	style := lipgloss.NewStyle()

	if sp.Color != nil {
		style = style.Foreground(lipgloss.Color(*sp.Color))
	}
	if sp.BackgroundColor != nil {
		style = style.Background(lipgloss.Color(*sp.BackgroundColor))
	}
	if sp.Bold != nil && *sp.Bold {
		style = style.Bold(true)
	}
	if sp.Italic != nil && *sp.Italic {
		style = style.Italic(true)
	}
	if sp.Underline != nil && *sp.Underline {
		style = style.Underline(true)
	}
	if sp.CrossedOut != nil && *sp.CrossedOut {
		style = style.Strikethrough(true)
	}

	return style
}
