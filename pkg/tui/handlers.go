package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

	"github.com/docker/cagent/pkg/browser"
	"github.com/docker/cagent/pkg/evaluation"
	mcptools "github.com/docker/cagent/pkg/tools/mcp"
	"github.com/docker/cagent/pkg/tui/components/editor"
	"github.com/docker/cagent/pkg/tui/components/notification"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/dialog"
	"github.com/docker/cagent/pkg/tui/messages"
	"github.com/docker/cagent/pkg/tui/page/chat"
	"github.com/docker/cagent/pkg/tui/service"
)

// Session management handlers

func (a *appModel) handleNewSession() (tea.Model, tea.Cmd) {
	a.application.NewSession()
	sess := a.application.Session()
	a.sessionState = service.NewSessionState(sess)
	a.chatPage = chat.New(a.application, a.sessionState)
	a.dialog = dialog.New()
	a.statusBar.SetHelp(a.chatPage)

	return a, tea.Batch(a.Init(), a.handleWindowResize(a.wWidth, a.wHeight))
}

func (a *appModel) handleOpenSessionBrowser() (tea.Model, tea.Cmd) {
	store := a.application.SessionStore()
	if store == nil {
		return a, notification.InfoCmd("No session store configured")
	}

	sessions, err := store.GetSessions(context.Background())
	if err != nil {
		return a, notification.ErrorCmd(fmt.Sprintf("Failed to load sessions: %v", err))
	}
	if len(sessions) == 0 {
		return a, notification.InfoCmd("No previous sessions found")
	}

	return a, core.CmdHandler(dialog.OpenDialogMsg{
		Model: dialog.NewSessionBrowserDialog(sessions),
	})
}

func (a *appModel) handleLoadSession(sessionID string) (tea.Model, tea.Cmd) {
	store := a.application.SessionStore()
	if store == nil {
		return a, notification.ErrorCmd("No session store configured")
	}

	sess, err := store.GetSession(context.Background(), sessionID)
	if err != nil {
		return a, notification.ErrorCmd(fmt.Sprintf("Failed to load session: %v", err))
	}

	// Cancel current session and replace with loaded one
	a.application.ReplaceSession(context.Background(), sess)
	a.sessionState = service.NewSessionState(sess)
	a.chatPage = chat.New(a.application, a.sessionState)
	a.dialog = dialog.New()
	a.statusBar.SetHelp(a.chatPage)

	return a, tea.Batch(a.Init(), a.handleWindowResize(a.wWidth, a.wHeight))
}

func (a *appModel) handleEvalSession(filename string) (tea.Model, tea.Cmd) {
	evalFile, _ := evaluation.Save(a.application.Session(), filename)
	return a, notification.SuccessCmd(fmt.Sprintf("Eval saved to file %s", evalFile))
}

func (a *appModel) handleCompactSession(additionalPrompt string) (tea.Model, tea.Cmd) {
	return a, a.chatPage.CompactSession(additionalPrompt)
}

func (a *appModel) handleCopySessionToClipboard() (tea.Model, tea.Cmd) {
	transcript := a.application.PlainTextTranscript()
	if transcript == "" {
		return a, notification.SuccessCmd("Conversation is empty; nothing copied.")
	}

	return a, tea.Sequence(
		tea.SetClipboard(transcript),
		func() tea.Msg {
			_ = clipboard.WriteAll(transcript)
			return nil
		},
		notification.SuccessCmd("Conversation copied to clipboard."),
	)
}

// Agent management handlers

func (a *appModel) handleSwitchAgent(agentName string) (tea.Model, tea.Cmd) {
	if err := a.application.SwitchAgent(agentName); err != nil {
		return a, notification.ErrorCmd(fmt.Sprintf("Failed to switch to agent '%s': %v", agentName, err))
	}

	a.currentAgent = agentName
	a.sessionState.SetCurrentAgent(agentName)
	return a, notification.SuccessCmd(fmt.Sprintf("Switched to agent '%s'", agentName))
}

func (a *appModel) handleToggleYolo() (tea.Model, tea.Cmd) {
	sess := a.application.Session()
	sess.ToolsApproved = !sess.ToolsApproved
	a.sessionState.SetYoloMode(sess.ToolsApproved)
	return a, nil
}

func (a *appModel) handleToggleHideToolResults() (tea.Model, tea.Cmd) {
	updated, cmd := a.chatPage.Update(messages.ToggleHideToolResultsMsg{})
	a.chatPage = updated.(chat.Page)
	return a, cmd
}

// MCP prompt handlers

func (a *appModel) handleShowMCPPromptInput(promptName string, promptInfo any) (tea.Model, tea.Cmd) {
	info, ok := promptInfo.(mcptools.PromptInfo)
	if !ok {
		return a, notification.ErrorCmd("Invalid prompt info")
	}

	return a, core.CmdHandler(dialog.OpenDialogMsg{
		Model: dialog.NewMCPPromptInputDialog(promptName, info),
	})
}

func (a *appModel) handleMCPPrompt(promptName string, arguments map[string]string) (tea.Model, tea.Cmd) {
	promptContent, err := a.application.ExecuteMCPPrompt(context.Background(), promptName, arguments)
	if err != nil {
		return a, notification.ErrorCmd(fmt.Sprintf("Error executing MCP prompt '%s': %v", promptName, err))
	}

	return a, core.CmdHandler(editor.SendMsg{Content: promptContent})
}

// Miscellaneous handlers

func (a *appModel) handleOpenURL(url string) (tea.Model, tea.Cmd) {
	_ = browser.Open(context.Background(), url)
	return a, nil
}

func (a *appModel) handleAgentCommand(command string) (tea.Model, tea.Cmd) {
	resolvedCommand := a.application.ResolveCommand(context.Background(), command)
	return a, core.CmdHandler(editor.SendMsg{Content: resolvedCommand})
}
