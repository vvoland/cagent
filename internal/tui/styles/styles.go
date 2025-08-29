package styles

import (
	"github.com/charmbracelet/lipgloss/v2"
)

// Color scheme - centralized color palette
var (
	// Primary colors
	highlight = lipgloss.Color("#7D56F4") // Purple for headers and focus

	// Status colors
	success    = lipgloss.Color("#00FF00") // Green for success/completed
	errorColor = lipgloss.Color("#FF0000") // Red for errors
	warning    = lipgloss.Color("#FFFF00") // Yellow for warnings/confirmation
	active     = lipgloss.Color("#00ff00") // Green for active/working states

	// Text colors
	muted     = lipgloss.Color("#808080") // Grey for muted text
	subtle    = lipgloss.Color("#C0C0C0") // Light grey for subtle text
	secondary = lipgloss.Color("#606060") // Darker grey for secondary text

	// Border colors
	borderPrimary = lipgloss.Color("#FFA500") // Orange for primary borders

	// State colors
	pending    = lipgloss.Color("#FFFFFF") // White for pending states
	inProgress = lipgloss.Color("#FFA500") // Orange for in-progress states
)

// Generic, reusable styles
var (
	// Base application style
	AppStyle = lipgloss.NewStyle().
			Padding(0, 1, 0, 1)

	// Text styles
	HighlightStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight)

	MutedStyle = lipgloss.NewStyle().
			Foreground(muted)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(subtle)

	SecondaryStyle = lipgloss.NewStyle().
			Foreground(secondary)

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(success)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	WarningStyle = lipgloss.NewStyle().
			Foreground(warning)

	ActiveStyle = lipgloss.NewStyle().
			Foreground(active)

	InProgressStyle = lipgloss.NewStyle().
			Foreground(inProgress)

	PendingStyle = lipgloss.NewStyle().
			Foreground(pending)

	// Layout styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			Padding(0, 0, 1, 0)

	BaseStyle = lipgloss.NewStyle()

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderPrimary)

	ToolsStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderPrimary).
			PaddingLeft(0).
			MarginLeft(0).
			Align(lipgloss.Left)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			Padding(2, 0, 1, 0)

	FocusedStyle = lipgloss.NewStyle().
			Padding(2, 0, 1, 0)

	// Layout helpers
	CenterStyle = lipgloss.NewStyle().
			Align(lipgloss.Center, lipgloss.Center)

	// Deprecated styles (kept for backward compatibility)
	FooterStyle = BaseStyle
	StatusStyle = MutedStyle
	ActionStyle = SecondaryStyle
	ChatStyle   = BaseStyle
)
