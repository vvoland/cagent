package editor

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/docker/cagent/internal/tui/core"
	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/styles"
	"github.com/docker/cagent/internal/tui/util"
)

// SendMsg represents a message to send
type SendMsg struct {
	Content string
}

// Editor represents an input editor component
type Editor interface {
	util.Model
	layout.Sizeable
	layout.Focusable
	layout.Help
}

// editorCmp implements Editor
type editorCmp struct {
	textarea textarea.Model
	width    int
	height   int
}

// New creates a new editor component
func New() Editor {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Focus()
	ta.Prompt = "│ "
	ta.CharLimit = -1
	ta.SetWidth(50)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Disable newline insertion on Enter

	return &editorCmp{
		textarea: ta,
	}
}

// Init initializes the component
func (e *editorCmp) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages and updates the component state
func (e *editorCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.textarea.SetWidth(msg.Width - 2)
		return e, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			if !e.textarea.Focused() {
				return e, nil
			}
			value := e.textarea.Value()
			if value != "" {
				e.textarea.Reset()
				return e, util.CmdHandler(SendMsg{Content: value})
			}
			return e, nil // Don't pass Enter to textarea when we handle it
		case "ctrl+c":
			return e, tea.Quit
		}
	}

	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

// View renders the component
func (e *editorCmp) View() string {
	style := styles.InputStyle
	if e.textarea.Focused() {
		style = styles.FocusedStyle
	}

	return style.Render(e.textarea.View())
}

// SetSize sets the dimensions of the component
func (e *editorCmp) SetSize(width, height int) tea.Cmd {
	e.width = width
	e.height = height

	// Account for border and padding
	contentWidth := max(width, 10)

	e.textarea.SetWidth(contentWidth)
	e.textarea.SetHeight(1) // Always single line

	return nil
}

// GetSize returns the current dimensions
func (e *editorCmp) GetSize() (width, height int) {
	return e.width, e.height
}

// Focus gives focus to the component
func (e *editorCmp) Focus() tea.Cmd {
	return e.textarea.Focus()
}

// Blur removes focus from the component
func (e *editorCmp) Blur() tea.Cmd {
	e.textarea.Blur()
	return nil
}

// IsFocused returns whether the component is focused
func (e *editorCmp) IsFocused() bool {
	return e.textarea.Focused()
}

// Bindings returns key bindings for the component
func (e *editorCmp) Bindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send"),
		),
	}
}

// Help returns the help information
func (e *editorCmp) Help() help.KeyMap {
	return core.NewSimpleHelp(e.Bindings(), [][]key.Binding{e.Bindings()})
}
