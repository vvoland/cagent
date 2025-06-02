package root

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
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
)

// Message types
type responseMsg struct{ content string }
type errorMsg error

type model struct {
	viewport    viewport.Model
	content     string // rendered content
	rawContent  string // raw markdown content
	ready       bool
	rt          *runtime.Runtime
	sess        *session.Session
	renderer    *glamour.TermRenderer
	inputBuffer string
	err         error
	responseCh  chan string
}

func (m *model) Init() tea.Cmd {
	return nil
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
				ch <- fmt.Sprintf("\n> 🔧 **Tool Call**: `%s(%s)`\n", e.ToolCall.Function.Name, truncateWithEllipsis(e.ToolCall.Function.Arguments, 20))
			case *runtime.ToolCallResponseEvent:
				ch <- fmt.Sprintf("> ✅ **Completed**: `%s`\n", truncateWithEllipsis(e.Response, 20))
			case *runtime.AgentMessageEvent:
				ch <- fmt.Sprintf("\n%s\n", e.Message.Content)
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

func (m *model) renderContent() error {
	rendered, err := m.renderer.Render(m.rawContent)
	if err != nil {
		return err
	}
	m.content = rendered
	m.viewport.SetContent(m.content)
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if strings.TrimSpace(m.inputBuffer) == "" {
				return m, nil
			}

			// Store the input before clearing it
			input := m.inputBuffer
			m.inputBuffer = ""

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
			return m, tea.Batch(
				processStream(m.rt, m.sess, m.responseCh),
				readResponse(m.responseCh),
			)
		}

		switch msg.Type {
		case tea.KeyRunes:
			m.inputBuffer += string(msg.Runes)
		case tea.KeySpace:
			m.inputBuffer += " "
		case tea.KeyBackspace:
			if m.inputBuffer != "" {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(highlight).
				Width(msg.Width).
				Height(msg.Height - 4)
			m.viewport.MouseWheelEnabled = true
			m.viewport.YPosition = 0
			m.ready = true

			// Create a new renderer with the current viewport width
			var err error
			m.renderer, err = glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(msg.Width),
			)
			if err != nil {
				m.err = err
			}
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4

			// Update renderer with new width
			var err error
			m.renderer, err = glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(msg.Width),
			)
			if err != nil {
				m.err = err
			}
		}

		// Re-render content with new width
		if err := m.renderContent(); err != nil {
			m.err = err
		}

	case responseMsg:
		m.rawContent += msg.content
		if err := m.renderContent(); err != nil {
			m.err = err
		}
		m.viewport.GotoBottom()
		return m, readResponse(m.responseCh)

	case errorMsg:
		m.err = error(msg)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var status string
	if m.err != nil {
		status = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	} else {
		status = statusStyle.Render("🤖 Ready")
	}

	input := fmt.Sprintf(
		"\n%s %s",
		inputPromptStyle.Render(">"),
		m.inputBuffer,
	)

	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.viewport.View(),
		status,
		input,
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

	// Initialize with a default width, it will be updated when we get the window size
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120), // Default width that will be updated
	)
	if err != nil {
		return err
	}

	m := &model{
		rt:       rt,
		sess:     session.New(agents),
		renderer: renderer,
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err = p.Run()
	return err
}
