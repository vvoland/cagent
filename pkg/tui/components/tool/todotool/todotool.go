package todotool

import (
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/tui/styles"
)

func renderTodoIcon(status string) (string, lipgloss.Style) {
	switch status {
	case "pending":
		return "◯", styles.PendingStyle
	case "in-progress":
		return "◕", styles.InProgressStyle
	case "completed":
		return "✓", styles.MutedStyle
	default:
		return "?", styles.BaseStyle
	}
}

func renderTodoDescriptionStyle(status string) lipgloss.Style {
	switch status {
	case "pending":
		return styles.PendingStyle
	case "in-progress":
		return styles.InProgressStyle
	case "completed":
		return styles.MutedStyle.Strikethrough(true)
	default:
		return styles.BaseStyle
	}
}
