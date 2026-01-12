//go:build darwin

package commands

import (
	tea "charm.land/bubbletea/v2"

	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/messages"
)

func speakCommand() *Item {
	return &Item{
		ID:           "session.speak",
		Label:        "Speak",
		SlashCommand: "/speak",
		Description:  "Start speech-to-text transcription (press Enter or Escape to stop)",
		Category:     "Session",
		Execute: func(string) tea.Cmd {
			return core.CmdHandler(messages.StartSpeakMsg{})
		},
	}
}
