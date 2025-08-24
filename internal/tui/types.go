package tui

import "time"

// ToolCall represents an active or completed tool call
type ToolCall struct {
	ID          string
	Name        string
	Arguments   string
	IsActive    bool
	IsCompleted bool
	Response    string
	StartTime   time.Time
}

// Message types
// We define dedicated types to leverage Bubble Tea's type-based message routing.
// They remain unexported as they are internal to the TUI.
type (
	responseMsg     struct{ content string }
	errorMsg        error
	toolCallMsg     struct{ toolCall ToolCall }
	toolCompleteMsg struct {
		id       string
		response string
	}
	workStartMsg    struct{}
	workEndMsg      struct{}
	readResponseMsg struct{}
)
