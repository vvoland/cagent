package dialog

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/markdown"
	"github.com/docker/cagent/pkg/tui/components/tool"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

// TODO(rumpl): use the tool factory to render the tool in the confirmation dialog

type (
	RuntimeResumeMsg struct {
		Response runtime.ResumeType
	}
)

// ToolConfirmationResponse represents the user's response to tool confirmation
type ToolConfirmationResponse struct {
	Response string // "approve", "reject", or "approve-session"
}

type toolConfirmationDialog struct {
	width, height int
	msg           *runtime.ToolCallConfirmationEvent
	keyMap        toolConfirmationKeyMap
	sessionState  *service.SessionState
}

// SetSize implements [Dialog].
func (d *toolConfirmationDialog) SetSize(width, height int) tea.Cmd {
	d.width = width
	d.height = height
	return nil
}

// toolConfirmationKeyMap defines key bindings for tool confirmation dialog
type toolConfirmationKeyMap struct {
	Yes key.Binding
	No  key.Binding
	All key.Binding
}

// defaultToolConfirmationKeyMap returns default key bindings
func defaultToolConfirmationKeyMap() toolConfirmationKeyMap {
	return toolConfirmationKeyMap{
		Yes: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("Y", "approve"),
		),
		No: key.NewBinding(
			key.WithKeys("n", "N"),
			key.WithHelp("N", "reject"),
		),
		All: key.NewBinding(
			key.WithKeys("a", "A"),
			key.WithHelp("A", "approve all"),
		),
	}
}

// NewToolConfirmationDialog creates a new tool confirmation dialog
func NewToolConfirmationDialog(msg *runtime.ToolCallConfirmationEvent, sessionState *service.SessionState) Dialog {
	return &toolConfirmationDialog{
		msg:          msg,
		sessionState: sessionState,
		keyMap:       defaultToolConfirmationKeyMap(),
	}
}

// Init initializes the tool confirmation dialog
func (d *toolConfirmationDialog) Init() tea.Cmd {
	return nil
}

// Update handles messages for the tool confirmation dialog
func (d *toolConfirmationDialog) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		return d, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keyMap.Yes):
			return d, tea.Sequence(core.CmdHandler(CloseDialogMsg{}), core.CmdHandler(RuntimeResumeMsg{Response: runtime.ResumeTypeApprove}))
		case key.Matches(msg, d.keyMap.No):
			return d, tea.Sequence(core.CmdHandler(CloseDialogMsg{}), core.CmdHandler(RuntimeResumeMsg{Response: runtime.ResumeTypeReject}))
		case key.Matches(msg, d.keyMap.All):
			return d, tea.Sequence(core.CmdHandler(CloseDialogMsg{}), core.CmdHandler(RuntimeResumeMsg{Response: runtime.ResumeTypeApproveSession}))
		}

		if msg.String() == "ctrl+c" {
			return d, tea.Quit
		}
	}

	return d, nil
}

// View renders the tool confirmation dialog
func (d *toolConfirmationDialog) View() string {
	dialogWidth := d.width * 70 / 100

	// Content width (accounting for padding and borders)
	contentWidth := dialogWidth - 6

	dialogStyle := styles.DialogStyle.Width(dialogWidth)

	// Title
	titleStyle := styles.DialogTitleStyle.Width(contentWidth)
	title := titleStyle.Render("Tool Confirmation")

	// Separator
	separatorWidth := max(contentWidth-10, 20)
	separator := styles.DialogSeparatorStyle.
		Align(lipgloss.Center).
		Width(contentWidth).
		Render(strings.Repeat("─", separatorWidth))

	a := types.Message{
		ToolCall: d.msg.ToolCall,
	}
	view := tool.New(&a, markdown.NewRenderer(contentWidth), d.sessionState)
	view.SetSize(contentWidth, 0)
	argumentsSection := view.View()

	question := styles.DialogQuestionStyle.Width(contentWidth).Render("Do you want to allow this tool call?")
	options := styles.DialogOptionsStyle.Width(contentWidth).Render("[Y]es    [N]o    [A]ll (approve all tools this session)")

	// Combine all parts with proper spacing
	parts := []string{title, separator}

	if argumentsSection != "" {
		parts = append(parts, "", argumentsSection)
	}

	parts = append(parts, "", question, "", options)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return dialogStyle.Render(content)
}

// Position calculates the position to center the dialog
func (d *toolConfirmationDialog) Position() (row, col int) {
	dialogWidth := d.width * 70 / 100

	// Estimate dialog height based on content
	dialogHeight := 12 // Base height for title, tool name, question, and options
	if d.msg.ToolCall.Function.Arguments != "" {
		// Add height for arguments section
		// Rough estimation: 3 lines for header + arguments
		dialogHeight += 5
	}

	// Add height for todo preview section if todo-related tools
	if d.msg.ToolCall.Function.Name == builtin.ToolNameCreateTodos || d.msg.ToolCall.Function.Name == builtin.ToolNameCreateTodo && d.msg.ToolCall.Function.Arguments != "" {
		// Add height for preview section header and content
		// Rough estimation: 2 lines for header + variable lines for todos
		dialogHeight += 6
	}

	// Ensure dialog stays on screen
	row = max(0, (d.height-dialogHeight)/2)
	col = max(0, (d.width-dialogWidth)/2)
	return row, col
}
