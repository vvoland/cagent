package messages

import (
	"strings"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/internal/app"
	"github.com/docker/cagent/internal/tui/components/message"
	"github.com/docker/cagent/internal/tui/components/tool"
	"github.com/docker/cagent/internal/tui/core"
	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/types"
	"github.com/docker/cagent/internal/tui/util"
)

// Model represents a chat message list component
type Model interface {
	util.Model
	layout.Sizeable
	layout.Focusable
	layout.Help

	AddUserMessage(content string) tea.Cmd
	AddAssistantMessage() tea.Cmd
	AddSeparatorMessage() tea.Cmd
	AddOrUpdateToolCall(toolName, toolCallID, arguments string, status types.ToolStatus) tea.Cmd
	// AddToolResult adds a tool result message or updates existing tool call with result
	AddToolResult(toolName, toolCallID, result string, status types.ToolStatus) tea.Cmd
	AppendToLastMessage(agentName string, content string) tea.Cmd
	ClearMessages()
	ScrollToBottom() tea.Cmd
	FocusToolInConfirmation() tea.Cmd
}

// model implements Model
type model struct {
	renderer     *glamour.TermRenderer
	messages     []types.Message
	views        []util.HeightableModel
	width        int
	height       int
	focused      bool
	scrollOffset int // Current scroll position in lines
	totalHeight  int // Total height of all content
	app          *app.App
	toolFocused  tool.Model // Currently focused tool for confirmation
}

// New creates a new message list component
func New(a *app.App) Model {
	mlc := &model{
		messages:     make([]types.Message, 0),
		views:        make([]util.HeightableModel, 0),
		width:        80, // Default width
		height:       24, // Default height
		scrollOffset: 0,
		totalHeight:  0,
		app:          a,
	}

	return mlc
}

// Init initializes the component
func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Initialize all message views
	for _, view := range m.views {
		if cmd := view.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

// Update handles messages and updates the component state
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update component dimensions
		cmd := m.SetSize(msg.Width, msg.Height)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Ensure scroll position is still valid after resize
		maxScroll := max(0, m.totalHeight-m.height)
		if m.scrollOffset > maxScroll {
			m.scrollOffset = maxScroll
		}
		// Continue processing other updates after resize

	case tea.MouseWheelMsg:
		// Use a reasonable scroll amount (3 lines per wheel event)
		const mouseScrollAmount = 3

		// Use Button field for direction detection since Y is always positive on this system
		buttonStr := msg.Button.String()

		switch buttonStr {
		case "wheelup":
			for range mouseScrollAmount {
				m.scrollUp()
			}
		case "wheeldown":
			for range mouseScrollAmount {
				m.scrollDown()
			}
		default:
			// Fallback to Y value for systems that use it correctly
			if msg.Y < 0 {
				// Use smaller scroll amount for negative Y systems too
				for range min(-msg.Y, mouseScrollAmount) {
					m.scrollUp()
				}
			} else if msg.Y > 0 {
				// Use smaller scroll amount for positive Y systems too
				for range min(msg.Y, mouseScrollAmount) {
					m.scrollDown()
				}
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		// Handle scrolling keys regardless of focus
		switch msg.String() {
		case "up", "k":
			m.scrollUp()
			return m, nil
		case "down", "j":
			m.scrollDown()
			return m, nil
		case "pgup":
			m.scrollPageUp()
			return m, nil
		case "pgdown":
			m.scrollPageDown()
			return m, nil
		case "home":
			m.scrollToTop()
			return m, nil
		case "end":
			m.scrollToBottom()
			return m, nil
		}

		// If focused and a tool needs confirmation, handle that first
		if m.focused && m.toolFocused != nil {
			if updatedModel, cmd := m.toolFocused.Update(msg); cmd != nil {
				m.toolFocused = updatedModel.(tool.Model)
				return m, cmd
			} else {
				m.toolFocused = updatedModel.(tool.Model)
			}
			return m, nil
		}
	}

	// Forward updates to all message views (for spinner updates)
	for i, view := range m.views {
		updatedView, cmd := view.Update(msg)
		if updatedView != nil {
			m.views[i] = updatedView.(message.Model)
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the component
func (m *model) View() string {
	return m.renderVisibleViews()
}

// SetSize sets the dimensions of the component
func (m *model) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height

	// Ensure minimum width for renderer to prevent issues
	if width < 10 {
		width = 10
	}

	// Initialize or update markdown renderer to match available width
	if r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
		glamour.WithStandardStyle("dark"),
	); err == nil {
		m.renderer = r
	}

	return nil
}

// GetSize returns the current dimensions
func (m *model) GetSize() (width, height int) {
	return m.width, m.height
}

// Focus gives focus to the component
func (m *model) Focus() tea.Cmd {
	m.focused = true
	return nil
}

// Blur removes focus from the component
func (m *model) Blur() tea.Cmd {
	m.focused = false
	return nil
}

// IsFocused returns whether the component is focused
func (m *model) IsFocused() bool {
	return m.focused
}

// Bindings returns key bindings for the component
func (m *model) Bindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
	}
}

// Help returns the help information
func (m *model) Help() help.KeyMap {
	return core.NewSimpleHelp(m.Bindings(), [][]key.Binding{m.Bindings()})
}

// AddUserMessage adds a user message to the chat
func (m *model) AddUserMessage(content string) tea.Cmd {
	msg := types.Message{
		Type:    types.MessageTypeUser,
		Content: content,
	}
	m.messages = append(m.messages, msg)

	view := m.createMessageView(&msg)
	m.views = append(m.views, view)

	// Initialize the new view
	return view.Init()
}

// AddAssistantMessage adds an assistant message to the chat
func (m *model) AddAssistantMessage() tea.Cmd {
	msg := types.Message{
		Type: types.MessageTypeAssistant,
	}
	wasAtBottom := m.isAtBottom()
	m.messages = append(m.messages, msg)

	view := m.createMessageView(&msg)
	m.views = append(m.views, view)

	// Initialize the new view (this will start the spinner for empty assistant messages)
	var cmds []tea.Cmd
	if initCmd := view.Init(); initCmd != nil {
		cmds = append(cmds, initCmd)
	}
	if wasAtBottom {
		// Scroll to bottom synchronously after adding content
		m.scrollToBottom()
	}
	return tea.Batch(cmds...)
}

// AddSeparatorMessage adds a separator message to the chat
func (m *model) AddSeparatorMessage() tea.Cmd {
	m.removeLastEmptyAssistantMessage()
	msg := types.Message{
		Type: types.MessageTypeSeparator,
	}
	m.messages = append(m.messages, msg)

	view := m.createMessageView(&msg)
	m.views = append(m.views, view)

	return view.Init()
}

// AddOrUpdateToolCall adds a tool call or updates existing one with the given status
func (m *model) AddOrUpdateToolCall(toolName, toolCallID, arguments string, status types.ToolStatus) tea.Cmd {
	// First try to update existing tool by ID (more precise)
	for i := len(m.messages) - 1; i >= 0; i-- {
		msg := &m.messages[i]
		if msg.ToolCallID == toolCallID {
			msg.ToolStatus = status
			if arguments != "" {
				msg.Arguments = arguments // Update arguments if provided
			}
			// Update the corresponding view
			view := m.createToolCallView(msg)
			m.views[i] = view
			// Initialize the updated view
			return view.Init()
		}
	}

	// If not found by ID, check if we need to remove last empty assistant message
	m.removeLastEmptyAssistantMessage()

	// Create new tool call
	msg := types.Message{
		Type:       types.MessageTypeToolCall,
		ToolName:   toolName,
		ToolCallID: toolCallID,
		ToolStatus: status,
		Arguments:  arguments,
	}
	m.messages = append(m.messages, msg)

	view := m.createToolCallView(&msg)
	m.views = append(m.views, view)

	// Initialize the new view
	return view.Init()
}

// AddToolResult adds tool result to the most recent matching tool call
func (m *model) AddToolResult(toolName, toolCallID, result string, status types.ToolStatus) tea.Cmd {
	// Find the tool call with matching name and ID and update it with the result
	for i := len(m.messages) - 1; i >= 0; i-- {
		msg := &m.messages[i]
		if msg.ToolCallID == toolCallID {
			// Update the existing tool call message with the result content
			msg.Content = result
			msg.ToolStatus = status
			// Update the corresponding view
			view := m.createToolCallView(msg)
			m.views[i] = view
			// Initialize the updated view
			return view.Init()
		}
	}

	return nil
}

// AppendToLastMessage appends content to the last message (for streaming)
func (m *model) AppendToLastMessage(agentName, content string) tea.Cmd {
	if len(m.messages) == 0 {
		return nil
	}
	lastIdx := len(m.messages) - 1
	lastMsg := &m.messages[lastIdx]

	// Only append to assistant messages - if last message is not assistant, create new one
	if lastMsg.Type == types.MessageTypeAssistant {
		wasAtBottom := m.isAtBottom()
		lastMsg.Content += content
		// Update the corresponding view
		view := m.createMessageView(lastMsg)
		m.views[lastIdx] = view
		// Initialize the updated view (needed for spinner state changes)
		var cmds []tea.Cmd
		if initCmd := view.Init(); initCmd != nil {
			cmds = append(cmds, initCmd)
		}
		if wasAtBottom {
			// Scroll to bottom synchronously after content update
			m.scrollToBottom()
		}
		return tea.Batch(cmds...)
	} else {
		// Last message is not assistant (probably a tool), create new assistant message
		msg := types.Message{
			Type:    types.MessageTypeAssistant,
			Content: content,
			Sender:  agentName,
		}
		wasAtBottom := m.isAtBottom()
		m.messages = append(m.messages, msg)

		view := m.createMessageView(&msg)
		m.views = append(m.views, view)

		// Initialize the new view
		var cmds []tea.Cmd
		if initCmd := view.Init(); initCmd != nil {
			cmds = append(cmds, initCmd)
		}
		if wasAtBottom {
			// Scroll to bottom synchronously after adding content
			m.scrollToBottom()
		}
		return tea.Batch(cmds...)
	}
}

// ClearMessages clears all messages
func (m *model) ClearMessages() {
	m.messages = make([]types.Message, 0)
	m.views = make([]util.HeightableModel, 0)
	m.scrollOffset = 0
	m.totalHeight = 0
}

// ScrollToBottom scrolls to the bottom of the chat
func (m *model) ScrollToBottom() tea.Cmd {
	// Make this synchronous to prevent race conditions
	m.scrollToBottom()
	return nil
}

// Virtual list implementation methods

func (m *model) createToolCallView(msg *types.Message) tool.Model {
	view := tool.New(msg, m.app)
	view.SetRenderer(m.renderer)
	view.SetSize(m.width, 0) // Height will be calculated dynamically
	return view
}

// createMessageView creates a properly initialized MessageView
func (m *model) createMessageView(msg *types.Message) message.Model {
	view := message.New(msg)
	view.SetRenderer(m.renderer)
	view.SetSize(m.width, 0) // Height will be calculated dynamically
	return view
}

// calculateTotalHeight calculates the total height of all content
func (m *model) calculateTotalHeight() {
	m.totalHeight = 0

	for _, view := range m.views {
		height := view.Height(m.width)
		m.totalHeight += height
	}
}

// isAtBottom returns true if the viewport is scrolled to the bottom
func (m *model) isAtBottom() bool {
	// Always recalculate height before checking bottom position
	m.calculateTotalHeight()
	maxScroll := max(0, m.totalHeight-m.height)
	return m.scrollOffset >= maxScroll
}

// renderVisibleViews renders only the views that are currently visible
func (m *model) renderVisibleViews() string {
	if len(m.views) == 0 {
		return ""
	}

	var content strings.Builder
	currentLine := 0
	hasWrittenLine := false

	writeLine := func(line string) {
		if hasWrittenLine {
			content.WriteString("\n")
		}
		content.WriteString(line)
		hasWrittenLine = true
	}

	viewportStart := m.scrollOffset
	viewportEnd := m.scrollOffset + m.height

	for _, view := range m.views {
		// Get view height
		viewHeight := view.Height(m.width)

		itemStart := currentLine
		itemEnd := currentLine + viewHeight

		// If item is at least partially visible
		if itemEnd > viewportStart && itemStart < viewportEnd {
			// Add the view content
			viewContent := view.View()
			viewLines := strings.SplitSeq(viewContent, "\n")
			for line := range viewLines {
				if currentLine >= viewportStart && currentLine < viewportEnd {
					writeLine(line)
				}
				currentLine++
			}
		} else {
			// Item not visible, just advance the line counter
			currentLine += viewHeight
		}

		// Stop rendering if we're past the visible area
		if currentLine >= viewportEnd {
			break
		}
	}

	return content.String()
}

// Scrolling methods
const defaultScrollAmount = 3 // Number of lines to scroll per scroll action

func (m *model) scrollUp() {
	if m.scrollOffset > 0 {
		m.scrollOffset = max(0, m.scrollOffset-defaultScrollAmount)
	}
}

func (m *model) scrollDown() {
	maxScroll := max(0, m.totalHeight-m.height)
	if m.scrollOffset < maxScroll {
		m.scrollOffset = min(maxScroll, m.scrollOffset+defaultScrollAmount)
	}
}

func (m *model) scrollPageUp() {
	m.scrollOffset = max(0, m.scrollOffset-m.height)
}

func (m *model) scrollPageDown() {
	maxScroll := max(0, m.totalHeight-m.height)
	m.scrollOffset = min(maxScroll, m.scrollOffset+m.height)
}

func (m *model) scrollToTop() {
	m.scrollOffset = 0
}

func (m *model) scrollToBottom() {
	maxScroll := max(0, m.totalHeight-m.height)
	m.scrollOffset = maxScroll
}

// FocusToolInConfirmation finds and focuses the first tool that needs confirmation
func (m *model) FocusToolInConfirmation() tea.Cmd {
	// Find the tool that needs confirmation (search backwards to get the most recent)
	for i := len(m.messages) - 1; i >= 0; i-- {
		msg := m.messages[i]
		if msg.Type == types.MessageTypeToolCall && msg.ToolStatus == types.ToolStatusConfirmation {
			if i < len(m.views) {
				if toolView, ok := m.views[i].(tool.Model); ok {
					m.toolFocused = toolView
					return toolView.Focus()
				}
			}
		}
	}
	return nil
}

// removeLastEmptyAssistantMessage removes the last message if it's an assistant message with empty content
func (m *model) removeLastEmptyAssistantMessage() {
	if len(m.messages) > 0 {
		lastIdx := len(m.messages) - 1
		lastMessage := m.messages[lastIdx]

		// Check if last message is an assistant message with empty content
		if lastMessage.Type == types.MessageTypeAssistant && strings.Trim(lastMessage.Content, "\r\n\t ") == "" {
			// Remove the last message and its corresponding view
			m.messages = m.messages[:lastIdx]
			if len(m.views) > lastIdx {
				m.views = m.views[:lastIdx]
			}
		}
	}
}
