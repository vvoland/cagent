package message

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/styles"
	"github.com/docker/cagent/internal/tui/types"
	"github.com/docker/cagent/internal/tui/util"
)

// Model represents a view that can render a message
type Model interface {
	util.Model
	layout.Sizeable

	// Render returns the rendered content for this message view
	Render(width int) string
	// Height returns the height this view will take when rendered at the given width
	Height(width int) int
	// Message returns the underlying message
	Message() *types.Message
	// SetRenderer sets the markdown renderer
	SetRenderer(renderer *glamour.TermRenderer)
}

// messageModel implements Model
type messageModel struct {
	message  *types.Message
	renderer *glamour.TermRenderer
	width    int
	height   int
	focused  bool
	spinner  spinner.Model
}

// New creates a new message view
func New(msg *types.Message) Model {
	s := spinner.New()
	s.Spinner = spinner.Points

	return &messageModel{
		message: msg,
		width:   80, // Default width
		height:  1,  // Will be calculated
		focused: false,
		spinner: s,
	}
}

// Bubble Tea Model methods

// Init initializes the message view
func (mv *messageModel) Init() tea.Cmd {
	// Start spinner for empty assistant messages
	if mv.message.Type == types.MessageTypeAssistant && mv.message.Content == "" {
		return mv.spinner.Tick
	}
	return nil
}

// Update handles messages and updates the message view state
func (mv *messageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle spinner updates for empty assistant messages
	if mv.message.Type == types.MessageTypeAssistant && mv.message.Content == "" {
		var cmd tea.Cmd
		mv.spinner, cmd = mv.spinner.Update(msg)
		return mv, cmd
	}

	// Message views typically don't handle input directly
	// They're controlled by the parent MessageListCmp
	return mv, nil
}

// View renders the message view
func (mv *messageModel) View() string {
	return mv.Render(mv.width)
}

// MessageView specific methods

// Render renders the message view content
func (mv *messageModel) Render(width int) string {
	msg := mv.message
	switch msg.Type {
	case types.MessageTypeUser:
		if rendered, err := mv.renderer.Render("> " + msg.Content); err == nil {
			return strings.TrimRight(rendered, "\n\r\t ")
		}
		return msg.Content
	case types.MessageTypeAssistant:
		if msg.Content == "" {
			if rendered, err := mv.renderer.Render("> " + mv.spinner.View()); err == nil {
				return strings.TrimRight(rendered, "\n\r\t ")
			}
			return fmt.Sprintf("> %s", mv.spinner.View())
		}
		rendered, err := mv.renderer.Render(fmt.Sprintf("> %s: %s", msg.Sender, msg.Content))
		if err != nil {
			sender := msg.Sender
			if sender == "" {
				sender = "root"
			}
			return strings.TrimRight(fmt.Sprintf("%s: %s", sender, msg.Content), "\n\r\t ")
		}
		return strings.TrimRight(rendered, "\n\r\t ")
	case types.MessageTypeSeparator:
		return styles.MutedStyle.Render("•" + strings.Repeat("─", mv.width-3) + "•")
	default:
		return msg.Content
	}
}

// Height calculates the height needed for this message view
func (mv *messageModel) Height(width int) int {
	content := mv.Render(width)
	return strings.Count(content, "\n") + 1
}

// Message returns the underlying message
func (mv *messageModel) Message() *types.Message {
	return mv.message
}

// SetRenderer sets the markdown renderer
func (mv *messageModel) SetRenderer(renderer *glamour.TermRenderer) {
	mv.renderer = renderer
}

// Layout.Sizeable methods

// SetSize sets the dimensions of the message view
func (mv *messageModel) SetSize(width, height int) tea.Cmd {
	mv.width = width
	mv.height = height
	return nil
}

// GetSize returns the current dimensions
func (mv *messageModel) GetSize() (width, height int) {
	return mv.width, mv.height
}
