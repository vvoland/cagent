package commands

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/dialog"
)

// Session commands
type (
	NewSessionMsg             struct{}
	EvalSessionMsg            struct{}
	CompactSessionMsg         struct{}
	CopySessionToClipboardMsg struct{}
)

// Agent commands
type AgentCommandMsg struct {
	Command string
}

func BuiltInSessionCommands() []dialog.Command {
	return []dialog.Command{
		{
			ID:           "session.new",
			Label:        "New",
			SlashCommand: "/new",
			Description:  "Start a new conversation",
			Category:     "Session",
			Execute: func() tea.Cmd {
				return core.CmdHandler(NewSessionMsg{})
			},
		},
		{
			ID:           "session.compact",
			Label:        "Compact",
			SlashCommand: "/compact",
			Description:  "Summarize the current conversation",
			Category:     "Session",
			Execute: func() tea.Cmd {
				return core.CmdHandler(CompactSessionMsg{})
			},
		},
		{
			ID:           "session.clipboard",
			Label:        "Copy",
			SlashCommand: "/copy",
			Description:  "Copy the current conversation to the clipboard",
			Category:     "Session",
			Execute: func() tea.Cmd {
				return core.CmdHandler(CopySessionToClipboardMsg{})
			},
		},
		{
			ID:           "session.eval",
			Label:        "Eval",
			SlashCommand: "/eval",
			Description:  "Create an evaluation report for the current conversation",
			Category:     "Session",
			Execute: func() tea.Cmd {
				return core.CmdHandler(EvalSessionMsg{})
			},
		},
	}
}

// BuildCommandCategories builds the list of command categories for the command palette
func BuildCommandCategories(ctx context.Context, application *app.App) []dialog.CommandCategory {
	categories := []dialog.CommandCategory{
		{
			Name:     "Session",
			Commands: BuiltInSessionCommands(),
		},
	}

	// Add agent commands if available
	agentCommands := application.CurrentAgentCommands(ctx)
	if len(agentCommands) > 0 {
		commands := make([]dialog.Command, 0, len(agentCommands))
		for name, prompt := range agentCommands {
			cmdText := "/" + name

			// Truncate long descriptions to fit on one line
			description := prompt
			if len(description) > 60 {
				description = description[:57] + "..."
			}

			// Capture cmdText in closure properly
			commandText := cmdText
			commands = append(commands, dialog.Command{
				ID:          "agent.command." + name,
				Label:       commandText,
				Description: description,
				Category:    "Agent Commands",
				Execute: func() tea.Cmd {
					return core.CmdHandler(AgentCommandMsg{Command: commandText})
				},
			})
		}

		categories = append(categories, dialog.CommandCategory{
			Name:     "Agent Commands",
			Commands: commands,
		})
	}

	return categories
}
