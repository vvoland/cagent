package util

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

// Model is the base interface for all TUI models
type Model interface {
	tea.Model
	tea.ViewModel
}

type HeightableModel interface {
	tea.Model
	tea.ViewModel
	Heightable
}

type Heightable interface {
	Height(width int) int
}

// CmdHandler creates a command that returns the given message
func CmdHandler(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}
