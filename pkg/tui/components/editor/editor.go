package editor

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/docker/cagent/pkg/history"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/styles"
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
	SetHistory(history *history.History)
	SetWorking(working bool) tea.Cmd
}

// editor implements Editor
type editor struct {
	textarea *textarea.Model
	width    int
	height   int
	working  bool

	history         *history.History
	draftInput      string
	historyBrowsing bool
}

// New creates a new editor component
func New() Editor {
	ta := textarea.New()
	ta.SetStyles(styles.InputStyle)
	ta.Placeholder = "Type your message here..."
	ta.Prompt = "│ "
	ta.CharLimit = -1
	ta.SetWidth(50)
	ta.SetHeight(3) // Set minimum 3 lines for multi-line input
	ta.Focus()
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
			if value != "" && !e.working {
				// Treat enter as send: clear input and exit history browse state.
				e.textarea.Reset()
				e.endHistoryBrowse()
				return e, core.CmdHandler(SendMsg{Content: value})
			}
			return e, nil
		case "ctrl+c":
			return e, tea.Quit
		case "up":
			// Step backward through command history when browsing.
			if e.navigateHistory(true) {
				return e, nil
			}
		case "down":
			// Step forward through command history when browsing.
			if e.navigateHistory(false) {
				return e, nil
			}
		default:
			// Any other key exits history browsing so input becomes fresh text.
			if e.historyBrowsing {
				e.endHistoryBrowse()
			}
		}
	}

	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

// View renders the component
func (e *editor) View() string {
	return styles.EditorStyle.Render(e.textarea.View())
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

// Bindings returns key bindings for the component
func (e *editor) Bindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send"),
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

func (e *editor) SetHistory(history *history.History) {
	e.history = history
}

func (e *editor) navigateHistory(previous bool) bool {
	if !e.canBrowseHistory() {
		return false
	}

	if !e.historyBrowsing {
		e.beginHistoryBrowse()
	}

	var entry string
	if previous {
		// Up arrow walks toward older commands.
		entry = e.history.Previous()
	} else {
		// Down arrow walks toward newer commands.
		entry = e.history.Next()
		if entry == "" {
			// Restore the draft when we step past the newest entry.
			e.restoreDraftFromHistory()
			return true
		}
	}

	if entry == "" {
		return true
	}

	// Replace the input with the selected history entry.
	e.textarea.SetValue(entry)
	e.textarea.MoveToEnd()
	return true
}

func (e *editor) canBrowseHistory() bool {
	return e.history != nil &&
		len(e.history.Messages) > 0 &&
		e.textarea.LineCount() == 1
}

func (e *editor) beginHistoryBrowse() {
	if e.history == nil {
		return
	}
	e.draftInput = e.textarea.Value()
	e.historyBrowsing = true
	e.moveHistoryCursorToLatest()
}

func (e *editor) restoreDraftFromHistory() {
	e.textarea.SetValue(e.draftInput)
	e.textarea.MoveToEnd()
	e.endHistoryBrowse()
}

func (e *editor) endHistoryBrowse() {
	e.historyBrowsing = false
	e.draftInput = ""
	if e.history == nil {
		return
	}
	e.moveHistoryCursorToLatest()
}

func (e *editor) moveHistoryCursorToLatest() {
	if e.history == nil {
		return
	}
	for e.history.Next() != "" {
	}
}
