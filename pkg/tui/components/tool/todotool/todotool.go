package todotool

import (
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/tui/styles"
)

// renderTodoIcon returns the icon and style for a todo status
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
