package toolcommon

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/paths"
	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

// ParseArgs unmarshals JSON arguments into a typed struct.
// Returns an error if parsing fails.
func ParseArgs[T any](args string) (T, error) {
	var result T
	var err error

	if err = json.Unmarshal([]byte(args), &result); err == nil {
		return result, nil
	}

	for _, fixed := range tryFixPartialJSON(args) {
		if partialErr := json.Unmarshal([]byte(fixed), &result); partialErr == nil {
			return result, nil
		}
	}

	return result, err
}

func tryFixPartialJSON(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s + "\"}", s + "}"}
}

// ExtractField creates an argument extractor function that parses JSON and extracts a field.
// The field function receives the parsed args and returns the display string.
// It supports partial JSON parsing for streaming tool calls.
func ExtractField[T any](field func(T) string) func(string) string {
	return func(args string) string {
		parsed, err := ParseArgs[T](args)
		if err != nil {
			return ""
		}
		return field(parsed)
	}
}

func Icon(msg *types.Message, inProgress spinner.Spinner) string {
	if msg.ToolStatus == types.ToolStatusPending || msg.ToolStatus == types.ToolStatusRunning {
		return styles.NoStyle.MarginLeft(2).Render(inProgress.View())
	}

	switch msg.ToolStatus {
	case types.ToolStatusCompleted:
		return styles.ToolCompletedIcon.Render("✓")
	case types.ToolStatusError:
		return styles.ToolErrorIcon.Render("✗")
	case types.ToolStatusConfirmation:
		return styles.ToolPendingIcon.Render("?")
	default:
		return styles.WarningStyle.Render("?")
	}
}

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

	availableWidth := max(width-styles.ToolCallResult.GetHorizontalFrameSize(), 10) // Minimum readable width

	lines := WrapLines(formattedContent, availableWidth)

	if len(lines) > 10 {
		lines = lines[:10]
		lines = append(lines, WrapLines("…", availableWidth)...)
	}

	return strings.Join(lines, "\n")
}

func RenderTool(msg *types.Message, inProgress spinner.Spinner, args, result string, width int, hideToolResults bool) string {
	nameStyle := styles.ToolName
	resultStyle := styles.ToolMessageStyle
	if msg.ToolStatus == types.ToolStatusError {
		nameStyle = styles.ToolNameError
		resultStyle = styles.ToolErrorMessageStyle
	}

	icon := Icon(msg, inProgress)
	name := nameStyle.Render(msg.ToolDefinition.DisplayName())
	content := fmt.Sprintf("%s%s", icon, name)

	if args != "" {
		firstLineWidth := width - lipgloss.Width(content) - 1 // -1 for space before args
		subsequentLineWidth := width - styles.ToolCompletedIcon.GetMarginLeft()
		wrappedArgs := wrapTextWithIndent(args, firstLineWidth, subsequentLineWidth)
		content += " " + wrappedArgs
	}
	if result != "" {
		if strings.Count(content, "\n") > 0 || strings.Count(result, "\n") > 0 {
			if !hideToolResults {
				content += "\n" + resultStyle.MarginLeft(styles.ToolCompletedIcon.GetMarginLeft()).Render(result)
			}
		} else {
			remainingWidth := max(width-lipgloss.Width(content)-1, 1)
			renderedResult := resultStyle.Render(result)
			if lipgloss.Width(renderedResult) > remainingWidth {
				// Truncate result to fit, leaving space for ellipsis
				renderedResult = resultStyle.Render(TruncateText(result, remainingWidth))
			}
			content += " " + lipgloss.PlaceHorizontal(remainingWidth, lipgloss.Right, renderedResult)
		}
	}

	return styles.RenderComposite(styles.ToolMessageStyle.Width(width), content)
}

// WrapLines wraps text to fit within the given width.
// Each line that exceeds the width is split at rune boundaries.
func WrapLines(text string, width int) []string {
	if width <= 0 {
		return strings.Split(text, "\n")
	}

	var lines []string
	for line := range strings.SplitSeq(text, "\n") {
		for lipgloss.Width(line) > width {
			breakPoint := findBreakPoint(line, width)
			runes := []rune(line)
			lines = append(lines, string(runes[:breakPoint]))
			line = string(runes[breakPoint:])
		}
		lines = append(lines, line)
	}
	return lines
}

// wrapTextWithIndent wraps text where the first line has a different available width.
// Subsequent lines are indented to align with the tool name badge.
// If text starts with a newline, it's considered pre-formatted and no indent is added.
func wrapTextWithIndent(text string, firstLineWidth, subsequentLineWidth int) string {
	if firstLineWidth <= 0 || subsequentLineWidth <= 0 {
		return text
	}

	// Pre-formatted text (starts with newline) doesn't need additional indentation
	if strings.HasPrefix(text, "\n") {
		return text
	}

	indent := strings.Repeat(" ", styles.ToolCompletedIcon.GetMarginLeft())
	var resultLines []string
	isFirstLine := true

	for inputLine := range strings.SplitSeq(text, "\n") {
		line := inputLine
		for line != "" {
			width := subsequentLineWidth
			prefix := indent
			if isFirstLine {
				width = firstLineWidth
				prefix = ""
			}

			if lipgloss.Width(line) <= width {
				resultLines = append(resultLines, prefix+line)
				break
			}

			// Find break point that fits within width
			breakPoint := findBreakPoint(line, width)
			runes := []rune(line)
			resultLines = append(resultLines, prefix+string(runes[:breakPoint]))
			line = string(runes[breakPoint:])
			isFirstLine = false
		}
		if inputLine == "" {
			resultLines = append(resultLines, indent)
		}
		isFirstLine = false
	}

	return strings.Join(resultLines, "\n")
}

// findBreakPoint finds the maximum number of runes that fit within the given width.
func findBreakPoint(line string, width int) int {
	runes := []rune(line)
	breakPoint := len(runes)
	for breakPoint > 0 && lipgloss.Width(string(runes[:breakPoint])) > width {
		breakPoint--
	}
	return max(breakPoint, 1) // At least one rune per line
}

// ShortenPath replaces home directory with ~ for cleaner display.
func ShortenPath(path string) string {
	if path == "" {
		return path
	}
	homeDir := paths.GetHomeDir()
	if homeDir != "" && strings.HasPrefix(path, homeDir) {
		return "~" + strings.TrimPrefix(path, homeDir)
	}
	return path
}

// TruncateText truncates text to fit within maxWidth, adding an ellipsis if needed.
// Uses lipgloss.Width for proper Unicode handling.
func TruncateText(text string, maxWidth int) string {
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	// Truncate by runes to handle Unicode properly
	runes := []rune(text)
	for lipgloss.Width(string(runes)) > maxWidth-1 && len(runes) > 0 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
