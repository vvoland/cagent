package toolcommon

import (
	"encoding/json"
	"strings"

	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

// Icon returns the status icon for a tool
func Icon(status types.ToolStatus) string {
	switch status {
	case types.ToolStatusPending:
		return "⊙"
	case types.ToolStatusRunning:
		return "⚙"
	case types.ToolStatusCompleted:
		return styles.SuccessStyle.Render("✓")
	case types.ToolStatusError:
		return styles.ErrorStyle.Render("✗")
	case types.ToolStatusConfirmation:
		return styles.WarningStyle.Render("?")
	default:
		return styles.WarningStyle.Render("?")
	}
}

// FormatToolResult formats tool result content for display.
// It handles JSON formatting and line wrapping, and truncates long output.
func FormatToolResult(content string, width int) string {
	var formattedContent string
	var m map[string]any
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		formattedContent = content
	} else if buf, err := json.MarshalIndent(m, "", "  "); err != nil {
		formattedContent = content
	} else {
		formattedContent = string(buf)
	}

	// Calculate available width for content (accounting for padding)
	padding := styles.ToolCallResult.Padding().GetHorizontalPadding()
	availableWidth := max(width-2-padding, 10) // Minimum readable width

	// Wrap long lines to fit the component width
	lines := wrapLines(formattedContent, availableWidth)

	header := "output"
	if len(lines) > 10 {
		lines = lines[:10]
		header = "output (truncated)"
		lines = append(lines, wrapLines("...", availableWidth)...)
	}

	// Join the lines back
	trimmedContent := strings.Join(lines, "\n")
	if trimmedContent != "" {
		return "\n" + styles.ToolCallResult.Render(styles.ToolCallResultKey.Render("\n-> "+header+":")+"\n"+trimmedContent)
	}

	return ""
}

// wrapLines wraps long lines to fit within the specified width
func wrapLines(text string, width int) []string {
	if width <= 0 {
		return strings.Split(text, "\n")
	}

	var lines []string

	for line := range strings.SplitSeq(text, "\n") {
		for len(line) > width {
			lines = append(lines, line[:width])
			line = line[width:]
		}

		lines = append(lines, line)
	}

	return lines
}
