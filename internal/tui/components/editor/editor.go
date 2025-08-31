package editor

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/docker/cagent/internal/tui/core"
	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/styles"
)

// SendMsg represents a message to send
type SendMsg struct {
	Content string
}

// Editor represents an input editor component
type Editor interface {
	layout.Model
	layout.Sizeable
	layout.Focusable
	layout.Help
	SetWorking(working bool) tea.Cmd
}

// editor implements Editor
type editor struct {
	textarea textarea.Model
	width    int
	height   int
	working  bool
}

// New creates a new editor component
func New() Editor {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Focus()
	ta.Prompt = "│ "
	ta.CharLimit = -1
	ta.SetWidth(50)
	ta.SetHeight(3) // Set minimum 3 lines for multi-line input
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(true) // Enable newline insertion

	return &editor{
		textarea: ta,
	}
}

// Init initializes the component
func (e *editor) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages and updates the component state
func (e *editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.textarea.SetWidth(msg.Width - 2)
		return e, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+enter":
			if !e.textarea.Focused() {
				return e, nil
			}
			value := e.textarea.Value()
			if value != "" && !e.working {
				e.textarea.Reset()
				return e, core.CmdHandler(SendMsg{Content: value})
			}
			return e, nil
		case "ctrl+c":
			return e, tea.Quit
		}
	}

	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

// View renders the component
func (e *editor) View() string {
	style := styles.InputStyle
	if e.textarea.Focused() {
		style = styles.FocusedStyle
	}

	return style.Render(e.textarea.View())
}

// SetSize sets the dimensions of the component
func (e *editor) SetSize(width, height int) tea.Cmd {
	e.width = width
	e.height = height

	// Account for border and padding
	contentWidth := max(width, 10)
	contentHeight := max(height, 3) // Minimum 3 lines, but respect height parameter

	e.textarea.SetWidth(contentWidth)
	e.textarea.SetHeight(contentHeight)

	return nil
}

// GetSize returns the current dimensions
func (e *editor) GetSize() (width, height int) {
	return e.width, e.height
}

// Focus gives focus to the component
func (e *editor) Focus() tea.Cmd {
	return e.textarea.Focus()
}

// Blur removes focus from the component
func (e *editor) Blur() tea.Cmd {
	e.textarea.Blur()
	return nil
}

// IsFocused returns whether the component is focused
func (e *editor) IsFocused() bool {
	return e.textarea.Focused()
}

// Bindings returns key bindings for the component
func (e *editor) Bindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("ctrl+enter"),
			key.WithHelp("ctrl+enter", "send"),
		),
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "new line"),
		),
	}
}

// Help returns the help information
func (e *editor) Help() help.KeyMap {
	return core.NewSimpleHelp(e.Bindings())
}

func (e *editor) SetWorking(working bool) tea.Cmd {
	e.working = working
	return nil
}
