package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#43BF6D"))

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(highlight)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	// Viewport styles
	chatViewportStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(highlight)

	toolViewportStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFA500")).
				PaddingLeft(0).
				MarginLeft(0).
				Align(lipgloss.Left)

	// Layout styles
	appStyle = lipgloss.NewStyle().
			Padding(1, 0, 0, 0)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight)

	footerStyle = lipgloss.NewStyle()

	// Tool call styles
	toolCallStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true)

	toolCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#43BF6D"))
)
