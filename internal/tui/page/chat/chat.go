package chat

import (
	"context"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/docker/cagent/internal/app"
	"github.com/docker/cagent/internal/tui/components/editor"
	"github.com/docker/cagent/internal/tui/components/messages"
	"github.com/docker/cagent/internal/tui/components/sidebar"
	"github.com/docker/cagent/internal/tui/core"
	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/dialog"

	"github.com/docker/cagent/internal/tui/styles"
	"github.com/docker/cagent/internal/tui/types"
	"github.com/docker/cagent/pkg/runtime"
)

// FocusedPanel represents which panel is currently focused
type FocusedPanel string

const (
	PanelSidebar FocusedPanel = "sidebar"
	PanelChat    FocusedPanel = "chat"
	PanelEditor  FocusedPanel = "editor"
)

// Page represents the main chat page
type Page interface {
	layout.Model
	layout.Sizeable
	layout.Help
}

// chatPage implements Page
type chatPage struct {
	width, height int
	sessionTitle  string

	// Components
	sidebar  sidebar.Model
	messages messages.Model
	editor   editor.Editor

	// State
	focusedPanel FocusedPanel

	// Key map
	keyMap KeyMap

	app *app.App

	// Cached layout dimensions
	chatHeight  int
	inputHeight int
}

// KeyMap defines key bindings for the chat page
type KeyMap struct {
	Tab   key.Binding
	Quit  key.Binding
	Send  key.Binding
	Focus key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch focus"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		Focus: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "focus chat"),
		),
	}
}

// New creates a new chat page
func New(a *app.App) Page {
	return &chatPage{
		sidebar:      sidebar.New(),
		messages:     messages.New(a),
		editor:       editor.New(),
		focusedPanel: PanelEditor,
		app:          a,
		keyMap:       DefaultKeyMap(),
	}
}

// Init initializes the chat page
func (p *chatPage) Init() tea.Cmd {
	return tea.Batch(
		p.sidebar.Init(),
		p.messages.Init(),
		p.editor.Init(),
		p.editor.Focus(),
	)
}

// Update handles messages and updates the page state
func (p *chatPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := p.SetSize(msg.Width, msg.Height)
		// Also forward resize event to components to ensure they handle it
		var cmds []tea.Cmd
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Forward to sidebar component
		sidebarModel, sidebarCmd := p.sidebar.Update(msg)
		p.sidebar = sidebarModel.(sidebar.Model)
		if sidebarCmd != nil {
			cmds = append(cmds, sidebarCmd)
		}
		// Forward to chat component
		chatModel, chatCmd := p.messages.Update(msg)
		p.messages = chatModel.(messages.Model)
		if chatCmd != nil {
			cmds = append(cmds, chatCmd)
		}
		// Forward to editor component
		editorModel, editorCmd := p.editor.Update(msg)
		p.editor = editorModel.(editor.Editor)
		if editorCmd != nil {
			cmds = append(cmds, editorCmd)
		}
		return p, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, p.keyMap.Quit):
			return p, tea.Quit
		case key.Matches(msg, p.keyMap.Tab):
			p.switchFocus()
			return p, nil
		case key.Matches(msg, p.keyMap.Focus):
			p.setFocusToChat()
			return p, nil
		}

		// Route other keys to focused component
		switch p.focusedPanel {
		case PanelChat:
			var cmd tea.Cmd
			var model tea.Model
			model, cmd = p.messages.Update(msg)
			p.messages = model.(messages.Model)
			return p, cmd
		case PanelEditor:
			var cmd tea.Cmd
			var model tea.Model
			model, cmd = p.editor.Update(msg)
			p.editor = model.(editor.Editor)
			return p, cmd
		}

		return p, nil

	case tea.MouseWheelMsg:
		// Always forward mouse wheel events to the chat component for scrolling
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = p.messages.Update(msg)
		p.messages = model.(messages.Model)
		return p, cmd

	case editor.SendMsg:
		cmd := p.processMessage(msg.Content)
		return p, cmd

	// Runtime events
	case *runtime.UserMessageEvent:
		cmd := p.messages.AddUserMessage(msg.Message)
		return p, tea.Batch(cmd, p.messages.ScrollToBottom())
	case *runtime.StreamStartedEvent:
		spinnerCmd := p.sidebar.SetWorking(true)
		cmd := p.messages.AddAssistantMessage()
		return p, tea.Batch(cmd, p.messages.ScrollToBottom(), spinnerCmd)
	case *runtime.AgentChoiceEvent:
		cmd := p.messages.AppendToLastMessage(msg.AgentName, msg.Choice.Delta.Content)
		return p, tea.Batch(cmd, p.messages.ScrollToBottom())
	case *runtime.SessionTitleEvent:
		p.sessionTitle = msg.Title
		p.sidebar.SetTitle(msg.Title)
	case *runtime.TokenUsageEvent:
		p.sidebar.SetTokenUsage(msg.Usage)
	case *runtime.StreamStoppedEvent:
		spinnerCmd := p.sidebar.SetWorking(false)
		cmd := p.messages.AddSeparatorMessage()
		return p, tea.Batch(cmd, p.messages.ScrollToBottom(), spinnerCmd)
	case *runtime.PartialToolCallEvent:
		// When we first receive a tool call, show it immediately in pending state
		spinnerCmd := p.sidebar.SetWorking(true)
		cmd := p.messages.AddOrUpdateToolCall(msg.ToolCall.Function.Name, msg.ToolCall.ID, msg.ToolCall.Function.Arguments, types.ToolStatusPending)
		return p, tea.Batch(cmd, p.messages.ScrollToBottom(), spinnerCmd)
	case *runtime.ToolCallConfirmationEvent:
		spinnerCmd := p.sidebar.SetWorking(false) // Stop working indicator during confirmation
		cmd := p.messages.AddOrUpdateToolCall(msg.ToolCall.Function.Name, msg.ToolCall.ID, msg.ToolCall.Function.Arguments, types.ToolStatusConfirmation)

		// Open tool confirmation dialog
		dialogCmd := core.CmdHandler(dialog.OpenDialogMsg{
			Model: dialog.NewToolConfirmationDialog(msg.ToolCall.Function.Name, msg.ToolCall.Function.Arguments, p.app),
		})

		return p, tea.Batch(cmd, p.messages.ScrollToBottom(), spinnerCmd, dialogCmd)
	case *runtime.ToolCallEvent:
		spinnerCmd := p.sidebar.SetWorking(true)
		cmd := p.messages.AddOrUpdateToolCall(msg.ToolCall.Function.Name, msg.ToolCall.ID, msg.ToolCall.Function.Arguments, types.ToolStatusRunning)

		// Check if this is a todo-related tool call and update sidebar
		toolName := msg.ToolCall.Function.Name
		if toolName == "todo_write" || toolName == "create_todo" || toolName == "create_todos" ||
			toolName == "update_todo" || toolName == "list_todos" {
			if err := p.sidebar.SetTodoArguments(toolName, msg.ToolCall.Function.Arguments); err != nil {
				// Log error but don't fail the tool call
				// Could add logging here if needed
			}
		}

		return p, tea.Batch(cmd, p.messages.ScrollToBottom(), spinnerCmd)
	case *runtime.ToolCallResponseEvent:
		spinnerCmd := p.sidebar.SetWorking(true)
		// Update the tool call with the response content and completed status
		cmd := p.messages.AddToolResult(msg.ToolCall.Function.Name, msg.ToolCall.ID, msg.Response, types.ToolStatusCompleted)

		// Return focus to editor after tool execution completes
		p.setFocusToEditor()
		return p, tea.Batch(cmd, p.messages.ScrollToBottom(), spinnerCmd)
	}

	sidebarModel, sidebarCmd := p.sidebar.Update(msg)
	p.sidebar = sidebarModel.(sidebar.Model)
	if sidebarCmd != nil {
		cmds = append(cmds, sidebarCmd)
	}

	chatModel, chatCmd := p.messages.Update(msg)
	p.messages = chatModel.(messages.Model)
	if chatCmd != nil {
		cmds = append(cmds, chatCmd)
	}

	editorModel, editorCmd := p.editor.Update(msg)
	p.editor = editorModel.(editor.Editor)
	if editorCmd != nil {
		cmds = append(cmds, editorCmd)
	}

	return p, tea.Batch(cmds...)
}

// View renders the chat page
func (p *chatPage) View() string {
	// Header
	headerText := "cagent"
	header := styles.HeaderStyle.Render(headerText + " " + p.sessionTitle)

	// Main chat content area (without input)
	// Calculate chat width (85% of available width)
	innerWidth := p.width // subtract app style padding
	sidebarWidth := int(float64(innerWidth) * 0.15)
	chatWidth := innerWidth - sidebarWidth

	chatView := styles.ChatStyle.
		Height(p.chatHeight).
		Width(chatWidth).
		Render(p.messages.View())

	// Sidebar with explicit height constraint to prevent disappearing during scroll
	sidebarView := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(p.chatHeight).
		Align(lipgloss.Left, lipgloss.Top).
		Render(p.sidebar.View())

	// Create horizontal layout with chat content and sidebar
	bodyContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		chatView,
		sidebarView,
	)

	// Input field spans full width below everything
	input := p.editor.View()

	// Create a full-height layout with header, body, and input
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		bodyContent,
		input,
	)

	return styles.AppStyle.
		Height(p.height).
		Render(content)
}

// SetSize sets the dimensions of the chat page
func (p *chatPage) SetSize(width, height int) tea.Cmd {
	p.width = width
	p.height = height

	var cmds []tea.Cmd

	// Calculate heights accounting for padding
	headerHeight := 3 // header + top/bottom padding
	editorHeight := 3 // fixed 3 lines for multi-line input

	// Calculate available space, ensuring status bar remains visible
	availableHeight := height - headerHeight
	p.inputHeight = editorHeight + 2 // account for editor padding
	p.chatHeight = availableHeight - p.inputHeight

	// Account for horizontal padding in width
	innerWidth := width - 2 // subtract left/right padding

	// Calculate sidebar and main content widths (15% sidebar, 85% main)
	sidebarWidth := int(float64(innerWidth) * 0.15)
	mainWidth := innerWidth - sidebarWidth

	// Set component sizes
	cmds = append(cmds,
		p.messages.SetSize(mainWidth, p.chatHeight),
		p.sidebar.SetSize(sidebarWidth, p.chatHeight),
		p.editor.SetSize(innerWidth, editorHeight), // Use calculated editor height
	)

	return tea.Batch(cmds...)
}

// GetSize returns the current dimensions
func (p *chatPage) GetSize() (width, height int) {
	return p.width, p.height
}

// Bindings returns key bindings for the chat page
func (p *chatPage) Bindings() []key.Binding {
	bindings := []key.Binding{
		p.keyMap.Tab,
		p.keyMap.Quit,
	}

	// Add focused component bindings
	switch p.focusedPanel {
	case PanelChat:
		bindings = append(bindings, p.messages.Bindings()...)
	case PanelEditor:
		bindings = append(bindings, p.editor.Bindings()...)
	}

	return bindings
}

// Help returns help information
func (p *chatPage) Help() help.KeyMap {
	return core.NewSimpleHelp(p.Bindings(), [][]key.Binding{p.Bindings()})
}

// switchFocus cycles between the focusable panels
func (p *chatPage) switchFocus() {
	// Clear focus from current panel
	switch p.focusedPanel {
	case PanelChat:
		p.messages.Blur()
	case PanelEditor:
		p.editor.Blur()
	}

	// Move to next panel
	switch p.focusedPanel {
	case PanelChat:
		p.focusedPanel = PanelEditor
		p.editor.Focus()
	case PanelEditor:
		p.focusedPanel = PanelChat
		p.messages.Focus()
	}
}

// setFocusToChat directly sets focus to the chat panel
func (p *chatPage) setFocusToChat() {
	// Clear focus from current panel
	switch p.focusedPanel {
	case PanelChat:
		// Already focused on chat, nothing to do
		return
	case PanelEditor:
		p.editor.Blur()
	}

	// Set focus to chat panel
	p.focusedPanel = PanelChat
	p.messages.Focus()
}

// setFocusToEditor directly sets focus to the editor panel
func (p *chatPage) setFocusToEditor() {
	// Clear focus from current panel
	switch p.focusedPanel {
	case PanelEditor:
		// Already focused on editor, nothing to do
		return
	case PanelChat:
		p.messages.Blur()
	}

	// Set focus to editor panel
	p.focusedPanel = PanelEditor
	p.editor.Focus()
}

// processMessage processes a message with the runtime
func (p *chatPage) processMessage(content string) tea.Cmd {
	p.app.Run(context.Background(), content)

	return tea.Batch(
		p.messages.ScrollToBottom(),
	)
}
