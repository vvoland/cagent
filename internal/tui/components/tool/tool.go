package tool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss/v2"

	"github.com/docker/cagent/internal/app"
	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/types"
	"github.com/docker/cagent/internal/tui/util"
)

// Model represents a view that can render a message
type Model interface {
	util.Model
	layout.Sizeable
	util.Heightable

	// Message returns the underlying message
	Message() *types.Message
	// SetRenderer sets the markdown renderer
	SetRenderer(renderer *glamour.TermRenderer)
	// Focus sets focus on the tool for handling input
	Focus() tea.Cmd
	// Blur removes focus from the tool
	Blur()
}

// toolModel implements Model
type toolModel struct {
	message  *types.Message
	renderer *glamour.TermRenderer
	width    int
	height   int
	focused  bool
	spinner  spinner.Model
	app      *app.App
}

// GetSize implements Model.
func (mv *toolModel) GetSize() (width, height int) {
	return mv.width, mv.height
}

// SetSize implements Model.
func (mv *toolModel) SetSize(width, height int) tea.Cmd {
	mv.width = width
	mv.height = height
	return nil
}

// New creates a new message view
func New(msg *types.Message, a *app.App) Model {
	s := spinner.New()
	s.Spinner = spinner.Points

	return &toolModel{
		message: msg,
		width:   80, // Default width
		height:  1,  // Will be calculated
		focused: false,
		spinner: s,
		app:     a,
	}
}

// Bubble Tea Model methods

// Init initializes the message view
func (mv *toolModel) Init() tea.Cmd {
	// Start spinner for empty assistant messages or pending/running tools
	if (mv.message.Type == types.MessageTypeAssistant && mv.message.Content == "") ||
		(mv.message.Type == types.MessageTypeToolCall &&
			(mv.message.ToolStatus == types.ToolStatusPending || mv.message.ToolStatus == types.ToolStatusRunning)) {
		return mv.spinner.Tick
	}
	return nil
}

// Update handles messages and updates the message view state
func (mv *toolModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle spinner updates for empty assistant messages or pending/running tools
	if (mv.message.Type == types.MessageTypeAssistant && mv.message.Content == "") ||
		(mv.message.Type == types.MessageTypeToolCall &&
			(mv.message.ToolStatus == types.ToolStatusPending || mv.message.ToolStatus == types.ToolStatusRunning)) {
		var cmd tea.Cmd
		mv.spinner, cmd = mv.spinner.Update(msg)
		return mv, cmd
	}

	// Handle keyboard input when in confirmation mode and focused
	if mv.focused && mv.message.ToolStatus == types.ToolStatusConfirmation {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "y", "Y":
				if mv.app != nil {
					mv.app.Resume("approve")
				}
				return mv, nil
			case "n", "N":
				if mv.app != nil {
					mv.app.Resume("reject")
				}
				return mv, nil
			case "a", "A":
				if mv.app != nil {
					mv.app.Resume("approve-session")
				}
				return mv, nil
			}
		}
	}

	// Message views typically don't handle input directly
	// They're controlled by the parent MessageListCmp
	return mv, nil
}

// View renders the message view
func (mv *toolModel) View() string {
	return mv.Render(mv.width)
}

// MessageView specific methods

// Render renders the message view content
func (mv *toolModel) Render(width int) string {
	msg := mv.message
	var icon string

	base := lipgloss.NewStyle()
	// Define color styles
	greenStyle := base.Foreground(lipgloss.Color("#00FF00"))
	redStyle := base.Foreground(lipgloss.Color("#FF0000"))
	yellowStyle := base.Foreground(lipgloss.Color("#FFFF00"))

	switch msg.ToolStatus {
	case types.ToolStatusPending:
		icon = "⊙"
	case types.ToolStatusRunning:
		icon = "⚙"
	case types.ToolStatusCompleted:
		icon = greenStyle.Render("✓")
	case types.ToolStatusError:
		icon = redStyle.Render("✗")
	case types.ToolStatusConfirmation:
		icon = yellowStyle.Render("?")
	default:
		icon = yellowStyle.Render("?")
	}

	// Add spinner for pending and running tools
	var spinnerText string
	if msg.ToolStatus == types.ToolStatusPending || msg.ToolStatus == types.ToolStatusRunning {
		spinnerText = " " + mv.spinner.View()
	}

	content := fmt.Sprintf("│ %s %s%s", icon, base.Bold(true).Render(msg.ToolName), spinnerText)

	confirmationContent := ""
	// Add confirmation options if in confirmation mode
	if msg.ToolStatus == types.ToolStatusConfirmation {
		var arguments map[string]any
		if err := json.Unmarshal([]byte(msg.Arguments), &arguments); err != nil {
			return ""
		}
		if len(arguments) > 0 {
			confirmationContent += "\n\nArguments:\n"
			for k, v := range arguments {
				confirmationContent += fmt.Sprintf("\n%s: %v", k, v)
			}
		}
		confirmationContent += "\n\nDo you want to allow this tool call?"
		confirmationContent += "\n**[Y]es** | **[N]o** | **[A]ll** (approve all tools this session)"
	}

	rendered, err := mv.renderer.Render(confirmationContent)
	if err != nil {
		return strings.TrimRight(content+confirmationContent, "\n\r\t")
	}
	return base.PaddingLeft(2).PaddingTop(2).Render(content + strings.TrimRight(rendered, "\n\r\t "))
}

// Height calculates the height needed for this message view
func (mv *toolModel) Height(width int) int {
	content := mv.Render(width)
	return strings.Count(content, "\n") + 1
}

// Message returns the underlying message
func (mv *toolModel) Message() *types.Message {
	return mv.message
}

// SetRenderer sets the markdown renderer
func (mv *toolModel) SetRenderer(renderer *glamour.TermRenderer) {
	mv.renderer = renderer
}

// Focus sets focus on the tool for handling input
func (mv *toolModel) Focus() tea.Cmd {
	mv.focused = true
	return nil
}

// Blur removes focus from the tool
func (mv *toolModel) Blur() {
	mv.focused = false
}
