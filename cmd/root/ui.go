package root

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/history"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/spf13/cobra"
)

var (
	// Styles
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#43BF6D"))

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(highlight)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	// Add viewport style
	viewportStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(highlight)

	// Add layout styles
	appStyle = lipgloss.NewStyle().
			Padding(1, 0, 0, 0) // Only add padding to the top

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight)

	footerStyle = lipgloss.NewStyle()
)

// Message types
type (
	responseMsg  struct{ content string }
	errorMsg     error
	showInputMsg struct{}
)

// model represents the application state
type model struct {
	// UI components
	viewport  viewport.Model
	textInput textinput.Model
	renderer  *glamour.TermRenderer

	// Content state
	content    string // rendered content
	rawContent string // raw markdown content
	err        error

	// App state
	ready     bool
	showInput bool // tracks when it's safe to show the text input
	width     int  // terminal width
	height    int  // terminal height

	// Business logic
	rt         *runtime.Runtime
	sess       *session.Session
	responseCh chan string
	history    *history.History
}

// newModel creates and initializes a new model
func newModel(rt *runtime.Runtime, sess *session.Session) (*model, error) {
	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Enter your message..."
	ti.Focus()
	ti.CharLimit = 0
	ti.Prompt = inputPromptStyle.Render("> ")

	hist, err := history.New()
	if err != nil {
		return nil, err
	}

	// Create viewport with mouse wheel enabled
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3 // Number of lines to scroll for each mouse wheel event

	return &model{
		viewport:   vp,
		textInput:  ti,
		rt:         rt,
		sess:       sess,
		responseCh: make(chan string, 100),
		history:    hist,
	}, nil
}

func (m *model) updateDimensions(width, height int) {
	m.width = width
	m.height = height

	// Update viewport dimensions
	headerHeight := 1
	footerHeight := 3
	viewportHeight := height - headerHeight - footerHeight

	m.viewport.Width = width
	m.viewport.Height = viewportHeight
	m.viewport.Style = viewportStyle.
		Width(width).
		Height(viewportHeight)

	// Update text input width
	m.textInput.Width = width - 2

	// Update renderer width
	var err error
	m.renderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		m.err = err
	}
}

// renderContent renders the raw markdown content
func (m *model) renderContent() error {
	rendered, err := m.renderer.Render(m.rawContent)
	if err != nil {
		return err
	}
	m.content = rendered
	m.viewport.SetContent(m.content)
	return nil
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

// Helper function to truncate string with ellipsis
func truncateWithEllipsis(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func processStream(rt *runtime.Runtime, sess *session.Session, ch chan<- string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		first := true

		for event := range rt.RunStream(ctx, sess) {
			switch e := event.(type) {
			case *runtime.AgentChoiceEvent:
				if first {
					ch <- fmt.Sprintf("\n**%s**: ", rt.CurrentAgent().Name())
					first = false
				}
				ch <- e.Choice.Delta.Content
			case *runtime.ToolCallEvent:
				ch <- fmt.Sprintf("\n\n> 🔧 **Tool Call**: `%s(%s)`\n", e.ToolCall.Function.Name, truncateWithEllipsis(e.ToolCall.Function.Arguments, 20))
			case *runtime.ToolCallResponseEvent:
				ch <- fmt.Sprintf("> ✅ **Completed**: `%s`\n\n", truncateWithEllipsis(e.Response, 20))
			case *runtime.AgentMessageEvent:
				ch <- fmt.Sprintf("\n\n%s\n\n", e.Message.Content)
			case *runtime.ErrorEvent:
				close(ch)
				return errorMsg(e.Error)
			}
		}
		close(ch)
		return nil
	}
}

func readResponse(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		if msg, ok := <-ch; ok {
			return responseMsg{content: msg}
		}
		return nil
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case showInputMsg:
		m.showInput = true
		return m, nil

	case tea.WindowSizeMsg:
		if !m.ready {
			m.updateDimensions(msg.Width, msg.Height)
			m.ready = true

			// Add a delay before showing the input
			return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
				return showInputMsg{}
			})
		}

		m.updateDimensions(msg.Width, msg.Height)
		if err := m.renderContent(); err != nil {
			m.err = err
		}

	case tea.KeyMsg:
		if !m.showInput {
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyUp:
			m.textInput.SetValue(m.history.Previous())
			return m, nil
		case tea.KeyDown:
			m.textInput.SetValue(m.history.Next())
			return m, nil
		case tea.KeyEnter:
			if strings.TrimSpace(m.textInput.Value()) == "" {
				return m, nil
			}
			cmd := m.handleUserInput()
			return m, cmd
		}

	case responseMsg:
		m.rawContent += msg.content
		if err := m.renderContent(); err != nil {
			m.err = err
		}
		m.viewport.GotoBottom()
		return m, tea.Tick(time.Millisecond*10, func(t time.Time) tea.Msg {
			return readResponseMsg{}
		})

	case readResponseMsg:
		return m, readResponse(m.responseCh)

	case errorMsg:
		m.err = error(msg)
		return m, nil
	}

	// Handle viewport updates
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}

	// Handle textinput updates if input is shown
	if m.showInput {
		var tiCmd tea.Cmd
		m.textInput, tiCmd = m.textInput.Update(msg)
		if tiCmd != nil {
			cmds = append(cmds, tiCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleUserInput processes user input and returns appropriate commands
func (m *model) handleUserInput() tea.Cmd {
	// Store the input before clearing it
	input := m.textInput.Value()
	m.textInput.Reset()

	// Add message to history
	if err := m.history.Add(input); err != nil {
		m.err = err
	}

	// Add user message to raw content
	userMsg := fmt.Sprintf("\n**You**: %s\n", input)
	m.rawContent += userMsg
	if err := m.renderContent(); err != nil {
		m.err = err
	}
	m.viewport.GotoBottom()

	// Add message to session
	m.sess.Messages = append(m.sess.Messages, session.AgentMessage{
		Agent: m.rt.CurrentAgent(),
		Message: chat.Message{
			Role:    "user",
			Content: input,
		},
	})

	// Create a new channel for this response
	m.responseCh = make(chan string, 100)

	return tea.Batch(
		processStream(m.rt, m.sess, m.responseCh),
		readResponse(m.responseCh),
	)
}

// Additional message type for reading responses
type readResponseMsg struct{}

func (m *model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Build header
	header := headerStyle.Render("🤖 AI Chat")

	// Build main content area
	content := m.viewport.View()

	// Build footer with status and input
	var footer string
	if m.err != nil {
		footer = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	} else {
		status := statusStyle.Render("🤖 Ready\n")
		input := ""
		if m.showInput {
			input = "\n" + m.textInput.View() + "\n"
		}
		footer = footerStyle.Render(status + input)
	}

	// Combine all sections
	return appStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			content,
			footer,
		),
	)
}

// NewUICmd creates a new UI command
func NewUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Run the agent with a TUI",
		Long:  `Run the agent with a Terminal User Interface powered by Charm`,
		RunE:  runUICommand,
	}

	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "agent.yaml", "Path to the configuration file")
	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")

	return cmd
}

func runUICommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.Default()
	logger.Debug("Starting agent UI", "agent", agentName)

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}

	agents, err := config.Agents(ctx, configFile)
	if err != nil {
		return err
	}

	rt, err := runtime.New(cfg, logger, agents, agentName)
	if err != nil {
		return err
	}

	m, err := newModel(rt, session.New(agents))
	if err != nil {
		return err
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(), // Enable mouse support
	)

	_, err = p.Run()
	return err
}
