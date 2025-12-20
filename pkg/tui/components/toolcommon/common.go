package toolcommon

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

func Icon(msg *types.Message, inProgress spinner.Spinner) string {
	if msg.ToolStatus == types.ToolStatusPending || msg.ToolStatus == types.ToolStatusRunning {
		return inProgress.View()
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

	padding := styles.ToolCallResult.Padding().GetHorizontalPadding()
	availableWidth := max(width-5-padding, 10) // Minimum readable width

	lines := wrapLines(formattedContent, availableWidth)

	if len(lines) > 10 {
		lines = lines[:10]
		lines = append(lines, wrapLines("…", availableWidth)...)
	}

	trimmedContent := strings.Join(lines, "\n")
	if trimmedContent != "" {
		return styles.ToolCallResult.Render(styles.ToolCallResultKey.Render("\n" + trimmedContent))
	}

	return ""
}

func RenderTool(msg *types.Message, inProgress spinner.Spinner, args, result string, width int) string {
	nameStyle := styles.ToolName
	resultStyle := styles.ToolMessageStyle
	if msg.ToolStatus == types.ToolStatusError {
		nameStyle = styles.ToolNameError
		resultStyle = styles.ToolErrorMessageStyle
	}

	content := fmt.Sprintf("%s%s", Icon(msg, inProgress), nameStyle.Render(msg.ToolDefinition.DisplayName()))

	if args != "" {
		content += " " + args
	}
	if result != "" {
		if strings.Count(content, "\n") > 0 {
			content += "\n" + result
		} else {
			remainingWidth := width - lipgloss.Width(content) - 2
			content += " " + lipgloss.PlaceHorizontal(remainingWidth, lipgloss.Right, resultStyle.Render(result))
		}
	}

	return styles.RenderComposite(styles.ToolMessageStyle.Width(width-1), content)
}

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
