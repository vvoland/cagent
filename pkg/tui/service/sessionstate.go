package service

import (
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/tui/types"
)

// SessionState holds shared state across the TUI application.
// This provides a centralized location for state that needs to be
// accessible by multiple components.
type SessionState struct {
	splitDiffView    bool
	yoloMode         bool
	thinking         bool
	hideToolResults  bool
	previousMessage  *types.Message
	currentAgentName string
	sessionTitle     string
	availableAgents  []runtime.AgentDetails
}

func NewSessionState(sessionState *session.Session) *SessionState {
	return &SessionState{
		splitDiffView:   true,
		yoloMode:        sessionState.ToolsApproved,
		thinking:        sessionState.Thinking,
		hideToolResults: sessionState.HideToolResults,
		sessionTitle:    sessionState.Title,
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

func (s *SessionState) CurrentAgentName() string {
	return s.currentAgentName
}

func (s *SessionState) SetCurrentAgentName(currentAgentName string) {
	s.currentAgentName = currentAgentName
}

func (s *SessionState) PreviousMessage() *types.Message {
	return s.previousMessage
}

func (s *SessionState) SetPreviousMessage(previousMessage *types.Message) {
	s.previousMessage = previousMessage
}

func (s *SessionState) SessionTitle() string {
	return s.sessionTitle
}

func (s *SessionState) SetSessionTitle(sessionTitle string) {
	s.sessionTitle = sessionTitle
}

func (s *SessionState) AvailableAgents() []runtime.AgentDetails {
	return s.availableAgents
}

func (s *SessionState) SetAvailableAgents(availableAgents []runtime.AgentDetails) {
	s.availableAgents = availableAgents
}
