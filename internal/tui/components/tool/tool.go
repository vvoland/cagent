package tool

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/internal/app"
	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/styles"
	"github.com/docker/cagent/internal/tui/types"
)

// Model represents a view that can render a message
type Model interface {
	layout.Model
	layout.Sizeable
	layout.Heightable

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

// wrapLines wraps long lines to fit within the specified width
func wrapLines(text string, width int) []string {
	if width <= 0 {
		return strings.Split(text, "\n")
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if len(line) <= width {
			wrappedLines = append(wrappedLines, line)
		} else {
			// Split long line into multiple lines
			for len(line) > width {
				wrappedLines = append(wrappedLines, line[:width])
				line = line[width:]
			}
			if line != "" {
				wrappedLines = append(wrappedLines, line)
			}
		}
	}
	return wrappedLines
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

	// Use predefined styles

	switch msg.ToolStatus {
	case types.ToolStatusPending:
		icon = "⊙"
	case types.ToolStatusRunning:
		icon = "⚙"
	case types.ToolStatusCompleted:
		icon = styles.SuccessStyle.Render("✓")
	case types.ToolStatusError:
		icon = styles.ErrorStyle.Render("✗")
	case types.ToolStatusConfirmation:
		icon = styles.WarningStyle.Render("?")
	default:
		icon = styles.WarningStyle.Render("?")
	}

	// Add spinner for pending and running tools
	var spinnerText string
	if msg.ToolStatus == types.ToolStatusPending || msg.ToolStatus == types.ToolStatusRunning {
		spinnerText = " " + mv.spinner.View()
	}

	content := fmt.Sprintf("│ %s %s%s", icon, styles.HighlightStyle.Render(msg.ToolName), spinnerText)

	// Add tool result content if available (for completed tools with content)
	var resultContent string
	if (msg.ToolStatus == types.ToolStatusCompleted || msg.ToolStatus == types.ToolStatusError) && msg.Content != "" {
		// Calculate available width for content (accounting for padding and prefixes)
		// Base padding (2) + content prefix and spacing
		availableWidth := width - 6
		if availableWidth < 10 {
			availableWidth = 10 // Minimum readable width
		}

		// Wrap long lines to fit the component width
		lines := wrapLines(msg.Content, availableWidth)

		// Take only first 10 lines after wrapping
		if len(lines) > 10 {
			lines = lines[:10]
			// Add indicator that content was truncated
			lines = append(lines, "... (output truncated)")
		}

		// Join the lines back and apply muted style
		trimmedContent := strings.Join(lines, "\n")
		if trimmedContent != "" {
			resultContent = "\n" + styles.MutedStyle.Render(trimmedContent)
		}
	}

	return styles.BaseStyle.PaddingLeft(2).PaddingTop(2).Render(content + resultContent)
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
