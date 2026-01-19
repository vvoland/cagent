package service

import (
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/tui/types"
)

// SessionState holds shared state across the TUI application.
// This provides a centralized location for state that needs to be
// accessible by multiple components.
type SessionState struct {
	splitDiffView   bool
	yoloMode        bool
	thinking        bool
	hideToolResults bool
	previousMessage *types.Message
	currentAgent    string
}

func NewSessionState(sessionState *session.Session) *SessionState {
	return &SessionState{
		splitDiffView:   true,
		yoloMode:        sessionState.ToolsApproved,
		thinking:        sessionState.Thinking,
		hideToolResults: sessionState.HideToolResults,
	}
}

func (s *SessionState) SplitDiffView() bool {
	return s.splitDiffView
}

func (s *SessionState) ToggleSplitDiffView() {
	s.splitDiffView = !s.splitDiffView
}

func (s *SessionState) YoloMode() bool {
	return s.yoloMode
}

func (s *SessionState) SetYoloMode(yoloMode bool) {
	s.yoloMode = yoloMode
}

func (s *SessionState) Thinking() bool {
	return s.thinking
}

func (s *SessionState) SetThinking(thinking bool) {
	s.thinking = thinking
}

func (s *SessionState) HideToolResults() bool {
	return s.hideToolResults
}

func (s *SessionState) ToggleHideToolResults() {
	s.hideToolResults = !s.hideToolResults
}

func (s *SessionState) SetHideToolResults(hideToolResults bool) {
	s.hideToolResults = hideToolResults
}

func (s *SessionState) CurrentAgent() string {
	return s.currentAgent
}

func (s *SessionState) SetCurrentAgent(currentAgent string) {
	s.currentAgent = currentAgent
}

func (s *SessionState) PreviousMessage() *types.Message {
	return s.previousMessage
}

func (s *SessionState) SetPreviousMessage(previousMessage *types.Message) {
	s.previousMessage = previousMessage
}
