package toolcommon

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/pkg/tui/styles"
)

const maxPreviewLines = 10

// getLanguageFromPath extracts the file extension to use as language hint for syntax highlighting
func getLanguageFromPath(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return ""
	}
	// Remove the leading dot
	return strings.TrimPrefix(ext, ".")
}

func RenderFile(path, content string, renderer *glamour.TermRenderer) string {
	lines := strings.Split(content, "\n")

	// Get first 10 lines
	previewLines := lines
	var truncated bool
	if len(lines) > maxPreviewLines {
		previewLines = lines[:maxPreviewLines]
		truncated = true
	}

	preview := strings.Join(previewLines, "\n")

	lang := getLanguageFromPath(path)

	markdown := "```" + lang + "\n" + preview + "\n```"

	rendered, err := renderer.Render(markdown)
	if err != nil {
		rendered = preview
	}

	var output strings.Builder
	output.WriteString(rendered)

	if truncated {
		totalLines := len(lines)
		remainingLines := totalLines - maxPreviewLines
		output.WriteString("\n")
		output.WriteString(styles.MutedStyle.Render("... ("))
		output.WriteString(styles.MutedStyle.Render(fmt.Sprintf("%d", remainingLines)))
		output.WriteString(styles.MutedStyle.Render(" more lines)"))
	}

	return output.String()
}
