package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/docker/cagent/pkg/history"
	"github.com/docker/cagent/pkg/tui/components/completion"
	"github.com/docker/cagent/pkg/tui/components/editor/completions"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/styles"
)

// SendMsg represents a message to send
type SendMsg struct {
	Content string
}

// historyNavigation describes which direction we want to pull from history.
type historyNavigation int

const (
	NAVIGATEPREVIOUS historyNavigation = iota
	NAVIGATENEXT
)

// Editor represents an input editor component
type Editor interface {
	layout.Model
	layout.Sizeable
	layout.Focusable
	layout.Help
	SetHistory(hist *history.History)
	SetWorking(working bool) tea.Cmd
}

// editor implements [Editor]
type editor struct {
	textarea *textarea.Model
	width    int
	height   int
	working  bool

	// history is the shared command store backing up/down navigation.
	hist *history.History
	// draftInput holds the user's unsent text while they browse history.
	draftInput string
	// historyBrowsing marks that we're currently showing history entries.
	historyBrowsing bool
	// completionWord stores the word being completed
	completionWord string
	// completions are the available completions
	completions []completions.Completion
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
		textarea:    ta,
		completions: completions.Completions(),
	}
}

// Init initializes the component
func (e *editor) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages and updates the component state
func (e *editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.textarea.SetWidth(msg.Width - 2)
		return e, nil
	case completion.SelectedMsg:
		currentValue := e.textarea.Value()

		lastIdx := strings.LastIndex(currentValue, e.completionWord)

		if lastIdx >= 0 {
			newValue := currentValue[:lastIdx-1] + msg.Value + currentValue[lastIdx+len(e.completionWord):]
			e.textarea.SetValue(newValue)
			e.textarea.MoveToEnd()
		}

		return e, nil
	case completion.ClosedMsg:
		e.completionWord = ""
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
			// Consume the key when we replace the buffer with an older command.
			if e.navigateHistory(NAVIGATEPREVIOUS) {
				return e, nil
			}
		case "down":
			// Consume the key when we replace the buffer with a newer command.
			if e.navigateHistory(NAVIGATENEXT) {
				return e, nil
			}
		default:
			for _, completion := range e.completions {
				if msg.String() == completion.Trigger() {
					cmds = append(cmds, e.startCompletion(completion))
				}
			}

			// Any other key exits history browsing so input becomes fresh text.
			if e.historyBrowsing {
				e.endHistoryBrowse()
			}
		}
	}

	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyMsg.String() == "space" {
			e.completionWord = ""
			cmds = append(cmds, core.CmdHandler(completion.CloseMsg{}))
		}

		currentWord := e.textarea.Word()
		if strings.HasPrefix(currentWord, "@") {
			e.completionWord = currentWord[1:]
			cmds = append(cmds, core.CmdHandler(completion.QueryMsg{Query: e.completionWord}))
		} else {
			e.completionWord = ""
			cmds = append(cmds, core.CmdHandler(completion.CloseMsg{}))
		}
	}

	return e, tea.Batch(cmds...)
}

func (e *editor) startCompletion(c completions.Completion) tea.Cmd {
	return core.CmdHandler(completion.OpenMsg{
		Items: c.Items(),
	})
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

func (e *editor) SetHistory(hist *history.History) {
	e.hist = hist
}

func (e *editor) navigateHistory(direction historyNavigation) bool {
	// Returning true tells Update to stop Bubble Tea's default cursor handling,
	// because we've already replaced the textarea content for this key press.
	if !e.canBrowseHistory() {
		return false
	}

	if !e.historyBrowsing {
		e.beginHistoryBrowse()
	}

	var entry string
	switch direction {
	case NAVIGATEPREVIOUS:
		// Up arrow walks toward older commands.
		entry = e.hist.Previous()
	case NAVIGATENEXT:
		// Down arrow walks toward newer commands.
		entry = e.hist.Next()
		if entry == "" {
			// Restore the draft when we step past the newest entry.
			e.restoreDraftFromHistory()
			return true
		}
	default:
		return false
	}

	if entry == "" {
		return true
	}

	// Replace the input with the selected history entry.
	e.textarea.SetValue(entry)
	// Place the cursor at the end so the user can immediately append or send.
	e.textarea.MoveToEnd()
	return true
}

func (e *editor) canBrowseHistory() bool {
	// We only take over arrow keys when there's at least one history entry and
	// the textarea is a single line (multi-line inputs retain normal movement).
	return e.hist != nil && e.textarea.Value() == ""
}

func (e *editor) beginHistoryBrowse() {
	if e.hist == nil {
		return
	}
	// Capture the in-progress text so we can restore it after browsing.
	e.draftInput = e.textarea.Value()
	e.historyBrowsing = true
	// Start from the newest entry so the first "up" pulls the latest command.
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
	if e.hist == nil {
		return
	}
	e.moveHistoryCursorToLatest()
}

func (e *editor) moveHistoryCursorToLatest() {
	if e.hist == nil {
		return
	}
	// Advance until Next returns empty, which positions the cursor just after
	// the most recent saved command.
	for e.hist.Next() != "" {
	}
}
