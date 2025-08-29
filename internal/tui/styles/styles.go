package styles

import (
	"github.com/charmbracelet/lipgloss/v2"
)

// Color scheme
var (
	highlight = lipgloss.Color("#7D56F4")
)

// Common styles used throughout the TUI
var (
	AppStyle = lipgloss.NewStyle().
			Padding(0, 1, 0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			Padding(0, 0, 1, 0)

	FooterStyle = lipgloss.NewStyle()

	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))

	ActionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	ChatStyle = lipgloss.NewStyle()

	ToolsStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFA500")).
			PaddingLeft(0).
			MarginLeft(0).
			Align(lipgloss.Left)

	InputStyle = lipgloss.NewStyle().
			Padding(2, 0, 1, 0)

	FocusedStyle = lipgloss.NewStyle().
			Padding(2, 0, 1, 0)
)
