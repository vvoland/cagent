package dialog

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/styles"
)

type oauthAuthorizationDialog struct {
	width, height int
	serverURL     string
	app           *app.App
	keyMap        oauthAuthorizationKeyMap
}

// SetSize implements [Dialog].
func (d *oauthAuthorizationDialog) SetSize(width, height int) tea.Cmd {
	d.width = width
	d.height = height
	return nil
}

// oauthAuthorizationKeyMap defines key bindings for OAuth authorization confirmation dialog
type oauthAuthorizationKeyMap struct {
	Yes key.Binding
	No  key.Binding
}

// defaultOAuthAuthorizationKeyMap returns default key bindings
func defaultOAuthAuthorizationKeyMap() oauthAuthorizationKeyMap {
	return oauthAuthorizationKeyMap{
		Yes: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("Y", "authorize"),
		),
		No: key.NewBinding(
			key.WithKeys("n", "N"),
			key.WithHelp("N", "decline"),
		),
	}
}

// NewOAuthAuthorizationDialog creates a new OAuth authorization confirmation dialog
func NewOAuthAuthorizationDialog(serverURL string, appInstance *app.App) Dialog {
	return &oauthAuthorizationDialog{
		serverURL: serverURL,
		app:       appInstance,
		keyMap:    defaultOAuthAuthorizationKeyMap(),
	}
}

// Init initializes the OAuth authorization confirmation dialog
func (d *oauthAuthorizationDialog) Init() tea.Cmd {
	return nil
}

// Update handles messages for the OAuth authorization confirmation dialog
func (d *oauthAuthorizationDialog) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		return d, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keyMap.Yes):
			_ = d.app.ResumeElicitation(context.Background(), tools.ElicitationActionAccept, nil)
			return d, core.CmdHandler(CloseDialogMsg{})
		case key.Matches(msg, d.keyMap.No):
			_ = d.app.ResumeElicitation(context.Background(), tools.ElicitationActionDecline, nil)
			return d, core.CmdHandler(CloseDialogMsg{})
		}

		if msg.String() == "ctrl+c" {
			return d, tea.Quit
		}
	}

	return d, nil
}

// Position returns the dialog position (centered)
func (d *oauthAuthorizationDialog) Position() (row, col int) {
	dialogContent := d.View()
	return CenterPosition(d.width, d.height, lipgloss.Width(dialogContent), lipgloss.Height(dialogContent))
}

// View renders the OAuth authorization confirmation dialog
func (d *oauthAuthorizationDialog) View() string {
	// clamped width: ~60% of screen, bounded by [40, 90] and screen margin
	dialogWidth := d.width * 60 / 100
	if dialogWidth < 40 {
		dialogWidth = max(24, min(d.width-4, 40))
	}
	if dialogWidth > 90 {
		dialogWidth = min(90, d.width-4)
	}

	padX := 2
	padY := 1

	// Border takes one character on each side when present
	frameHorizontal := (padX * 2) + 2
	contentWidth := max(10, dialogWidth-frameHorizontal)

	dialogStyle := styles.DialogWarningStyle.
		Padding(padY, padX).
		Width(dialogWidth)

	title := styles.DialogTitleInfoStyle.
		Width(contentWidth).
		Render("🔐 OAuth Authorization Required")

	serverInfo := styles.InfoStyle.
		Width(contentWidth).
		Render(fmt.Sprintf("Server: %s (remote)", d.serverURL))

	description := styles.DialogContentStyle.
		Width(contentWidth).
		Render("This server requires OAuth authentication to access its tools. Your browser will open automatically to complete the authorization process.")

	instructions := styles.DialogHelpStyle.
		Width(contentWidth).
		Render("After authorizing in your browser, return here and the agent will continue automatically.")

	options := styles.SuccessStyle.
		Align(lipgloss.Center).
		Width(contentWidth).
		Render("Y - Authorize  |  N - Decline")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		serverInfo,
		"",
		description,
		"",
		instructions,
		"",
		options,
	)

	return dialogStyle.Render(content)
}
