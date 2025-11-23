package editfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

const (
	tabWidth     = 4
	lineNumWidth = 5
	minWidth     = 80
)

var (
	lexerCache   = make(map[string]chroma.Lexer)
	lexerCacheMu sync.RWMutex
)

type chromaToken struct {
	Text  string
	Style lipgloss.Style
}

type linePair struct {
	old        *udiff.Line
	new        *udiff.Line
	oldLineNum int
	newLineNum int
}

func renderEditFile(toolCall tools.ToolCall, width int, splitView bool, toolStatus types.ToolStatus) string {
	var args builtin.EditFileArgs
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return ""
	}

	var output strings.Builder
	for i, edit := range args.Edits {
		if i > 0 {
			output.WriteString("\n\n")
		}

		if len(args.Edits) > 1 {
			output.WriteString("Edit #" + string(rune(i+1+'0')) + ":\n")
		}

		diff := computeDiff(args.Path, edit.OldText, edit.NewText, toolStatus)
		if splitView {
			output.WriteString(renderSplitDiffWithSyntaxHighlight(diff, args.Path, width))
		} else {
			output.WriteString(renderDiffWithSyntaxHighlight(diff, args.Path, width))
		}
	}

	return output.String()
}

func computeDiff(path, oldText, newText string, toolStatus types.ToolStatus) []*udiff.Hunk {
	currentContent, err := os.ReadFile(path)
	if err != nil {
		return []*udiff.Hunk{}
	}

	var oldContent, newContent string

	if toolStatus == types.ToolStatusConfirmation {
		// During confirmation: file hasn't been modified yet
		// currentContent is the old content, we need to compute what new would be
		oldContent = string(currentContent)
		newContent = strings.Replace(oldContent, oldText, newText, 1)
	} else {
		// After execution: file has been modified
		// currentContent is the new content, we need to reconstruct old
		newContent = string(currentContent)
		oldContent = strings.Replace(newContent, newText, oldText, 1)
	}

	// Now compute diff between old and new
	edits := udiff.Strings(oldContent, newContent)

	diff, err := udiff.ToUnifiedDiff("old", "new", oldContent, edits, 3)
	if err != nil {
		return []*udiff.Hunk{}
	}

	return normalizeDiff(diff.Hunks)
}

func normalizeDiff(diff []*udiff.Hunk) []*udiff.Hunk {
	for _, hunk := range diff {
		if len(hunk.Lines) == 0 {
			continue
		}

		normalized := make([]udiff.Line, 0, len(hunk.Lines))
		for i := 0; i < len(hunk.Lines); i++ {
			line := hunk.Lines[i]

			if line.Kind == udiff.Delete && i+1 < len(hunk.Lines) {
				next := hunk.Lines[i+1]
				if next.Kind == udiff.Insert && line.Content == next.Content {
					normalized = append(normalized, udiff.Line{
						Kind:    udiff.Equal,
						Content: line.Content,
					})
					i++
					continue
				}
			}

			normalized = append(normalized, line)
		}

		hunk.Lines = normalized
	}

	return diff
}

func syntaxHighlight(code, filePath string) []chromaToken {
	ext := filepath.Ext(filePath)

	// Try to get lexer from cache
	lexerCacheMu.RLock()
	lexer, ok := lexerCache[ext]
	lexerCacheMu.RUnlock()

	if !ok {
		// Cache miss - compute and store
		lexer = lexers.Match(filePath)
		if lexer == nil {
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)

		lexerCacheMu.Lock()
		lexerCache[ext] = lexer
		lexerCacheMu.Unlock()
	}

	style := styles.ChromaStyle()
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return []chromaToken{{Text: code, Style: lipgloss.NewStyle()}}
	}

	var tokens []chromaToken
	for _, token := range iterator.Tokens() {
		if token.Value == "" {
			continue
		}
		tokens = append(tokens, chromaToken{
			Text:  token.Value,
			Style: chromaToLipgloss(token.Type, style),
		})
	}

	return tokens
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

func renderDiffWithSyntaxHighlight(diff []*udiff.Hunk, filePath string, width int) string {
	var output strings.Builder
	contentWidth := width - lineNumWidth

	for _, hunk := range diff {
		oldLineNum := hunk.FromLine
		newLineNum := hunk.ToLine

		for _, line := range hunk.Lines {
			lineNum := getDisplayLineNumber(&line, &oldLineNum, &newLineNum)
			content := prepareContent(line.Content, contentWidth)

			lineNumStr := styles.LineNumberStyle.Render(fmt.Sprintf("%4d ", lineNum))
			styledLine := renderLine(content, line.Kind, filePath, contentWidth)

			output.WriteString(lineNumStr + styledLine + "\n")
		}
	}

	return strings.TrimSuffix(output.String(), "\n")
}

func renderSplitDiffWithSyntaxHighlight(diff []*udiff.Hunk, filePath string, width int) string {
	// Fall back to unified diff if terminal is too narrow
	separator := styles.SeparatorStyle.Render(" │ ")
	separatorWidth := ansi.StringWidth(separator)
	contentWidth := (width - separatorWidth - (lineNumWidth * 2)) / 2

	if width < minWidth || contentWidth < 10 {
		return renderDiffWithSyntaxHighlight(diff, filePath, width)
	}

	var output strings.Builder

	for _, hunk := range diff {
		for _, pair := range pairDiffLines(hunk.Lines, hunk.FromLine, hunk.ToLine) {
			leftSide := renderSplitSide(pair.old, pair.oldLineNum, filePath, contentWidth)
			rightSide := renderSplitSide(pair.new, pair.newLineNum, filePath, contentWidth)

			line := leftSide + separator + rightSide
			line = ensureWidth(line, width)

			output.WriteString(line + "\n")
		}
	}

	return strings.TrimSuffix(output.String(), "\n")
}

func getDisplayLineNumber(line *udiff.Line, oldLineNum, newLineNum *int) int {
	switch line.Kind {
	case udiff.Delete:
		num := *oldLineNum
		*oldLineNum++
		return num
	case udiff.Insert:
		num := *newLineNum
		*newLineNum++
		return num
	case udiff.Equal:
		num := *oldLineNum
		*oldLineNum++
		*newLineNum++
		return num
	}
	return 0
}

func prepareContent(content string, maxWidth int) string {
	content = strings.ReplaceAll(content, "\t", strings.Repeat(" ", tabWidth))
	content = strings.TrimRight(content, "\n")
	if runewidth.StringWidth(content) > maxWidth {
		content = runewidth.Truncate(content, maxWidth-3, "...")
	}
	return content
}

func renderLine(content string, kind udiff.OpKind, filePath string, width int) string {
	tokens := syntaxHighlight(content, filePath)
	lineStyle := getLineStyle(kind)

	rendered := renderTokensWithStyle(tokens, lineStyle)

	return padToWidth(rendered, width, lineStyle)
}

func renderSplitSide(line *udiff.Line, lineNum int, filePath string, width int) string {
	lineNumStr := formatLineNum(line, lineNum)

	if line == nil {
		emptySpace := styles.DiffUnchangedStyle.Render(strings.Repeat(" ", width))
		return styles.LineNumberStyle.Render(lineNumStr) + emptySpace
	}

	content := prepareContent(line.Content, width)
	styledContent := renderLine(content, line.Kind, filePath, width)

	return styles.LineNumberStyle.Render(lineNumStr) + styledContent
}

func renderTokensWithStyle(tokens []chromaToken, lineStyle lipgloss.Style) string {
	var output strings.Builder

	for _, token := range tokens {
		styledToken := token.Style.Background(lineStyle.GetBackground())
		output.WriteString(styledToken.Render(token.Text))
	}

	return output.String()
}

func padToWidth(content string, width int, style lipgloss.Style) string {
	currentWidth := ansi.StringWidth(content)
	if paddingNeeded := width - currentWidth; paddingNeeded > 0 {
		padding := strings.Repeat(" ", paddingNeeded)
		return content + style.Render(padding)
	}
	return content
}

func ensureWidth(line string, width int) string {
	if lineWidth := ansi.StringWidth(line); lineWidth < width {
		padding := styles.DiffUnchangedStyle.Render(strings.Repeat(" ", width-lineWidth))
		return line + padding
	}
	return line
}

func getLineStyle(kind udiff.OpKind) lipgloss.Style {
	switch kind {
	case udiff.Delete:
		return styles.DiffRemoveStyle
	case udiff.Insert:
		return styles.DiffAddStyle
	default:
		return styles.DiffUnchangedStyle
	}
}

func formatLineNum(line *udiff.Line, lineNum int) string {
	if line == nil {
		return strings.Repeat(" ", lineNumWidth)
	}
	return fmt.Sprintf("%4d ", lineNum)
}

func pairDiffLines(lines []udiff.Line, fromLine, toLine int) []linePair {
	var pairs []linePair
	oldLineNum, newLineNum := fromLine, toLine

	for i := 0; i < len(lines); i++ {
		line := &lines[i]

		switch line.Kind {
		case udiff.Equal:
			pairs = append(pairs, linePair{
				old:        line,
				new:        line,
				oldLineNum: oldLineNum,
				newLineNum: newLineNum,
			})
			oldLineNum++
			newLineNum++

		case udiff.Delete:
			// Check if next line is an insert to pair them
			if i+1 < len(lines) && lines[i+1].Kind == udiff.Insert {
				pairs = append(pairs, linePair{
					old:        line,
					new:        &lines[i+1],
					oldLineNum: oldLineNum,
					newLineNum: newLineNum,
				})
				oldLineNum++
				newLineNum++
				i++ // Skip the paired insert
			} else {
				// Unpaired delete
				pairs = append(pairs, linePair{
					old:        line,
					new:        nil,
					oldLineNum: oldLineNum,
				})
				oldLineNum++
			}

		case udiff.Insert:
			// Unpaired insert (paired inserts are handled above)
			pairs = append(pairs, linePair{
				old:        nil,
				new:        line,
				newLineNum: newLineNum,
			})
			newLineNum++
		}
	}

	return pairs
}
