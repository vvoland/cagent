package service

// SessionState holds shared state across the TUI application.
// This provides a centralized location for state that needs to be
// accessible by multiple components.
type SessionState struct {
	// TodoManager manages the state of all todos across the application
	TodoManager *TodoManager

	// SplitDiffView determines whether diff views should be shown side-by-side (true)
	// or unified (false)
	SplitDiffView bool
}

// NewSessionState creates a new SessionState with default values.
func NewSessionState() *SessionState {
	return &SessionState{
		TodoManager:   NewTodoManager(),
		SplitDiffView: true, // Default to split view
	}
}

// ToggleSplitDiffView toggles between split and unified diff view modes.
func (s *SessionState) ToggleSplitDiffView() {
	s.SplitDiffView = !s.SplitDiffView
}
