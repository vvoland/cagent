package sidebar

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/components/tool/todotool"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
)

type Mode int

const (
	ModeVertical Mode = iota
	ModeHorizontal
)

// Model represents a sidebar component
type Model interface {
	layout.Model
	layout.Sizeable

	SetTokenUsage(usage *runtime.Usage)
	SetTodos(toolCall tools.ToolCall) error
	SetWorking(working bool) tea.Cmd
	SetMode(mode Mode)
	SetAgentInfo(agentName, model, description string)
	SetTeamInfo(availableAgents []string)
	SetAgentSwitching(switching bool)
	SetToolsetInfo(availableTools int)
	SetToolStatus(toolName, status string)
	GetSize() (width, height int)
}

// model implements Model
type model struct {
	width            int
	height           int
	usage            *runtime.Usage
	todoComp         *todotool.SidebarComponent
	working          bool
	mcpInit          bool
	spinner          spinner.Spinner
	mode             Mode
	sessionTitle     string
	currentAgent     string
	agentModel       string
	agentDescription string
	availableAgents  []string
	agentSwitching   bool
	availableTools   int
	activeTools      []string
	toolExecutions   map[string]string // tool name -> status (running, completed, failed)
}

func New(manager *service.TodoManager) Model {
	return &model{
		width:          20,
		height:         24,
		usage:          &runtime.Usage{},
		todoComp:       todotool.NewSidebarComponent(manager),
		spinner:        spinner.New(spinner.ModeSpinnerOnly),
		sessionTitle:   "New session",
		toolExecutions: make(map[string]string),
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) SetTokenUsage(usage *runtime.Usage) {
	m.usage = usage
}

func (m *model) SetTodos(toolCall tools.ToolCall) error {
	return m.todoComp.SetTodos(toolCall)
}

// SetWorking sets the working state and returns a command to start the spinner if needed
func (m *model) SetWorking(working bool) tea.Cmd {
	m.working = working
	if working {
		return m.spinner.Init()
	}
	return nil
}

// SetAgentInfo sets the current agent information
func (m *model) SetAgentInfo(agentName, model, description string) {
	m.currentAgent = agentName
	m.agentModel = model
	m.agentDescription = description
}

// SetTeamInfo sets the available agents in the team
func (m *model) SetTeamInfo(availableAgents []string) {
	m.availableAgents = availableAgents
}

// SetAgentSwitching sets whether an agent switch is in progress
func (m *model) SetAgentSwitching(switching bool) {
	m.agentSwitching = switching
}

// SetToolsetInfo sets the number of available tools
func (m *model) SetToolsetInfo(availableTools int) {
	m.availableTools = availableTools
}

// SetToolStatus updates the status of a specific tool
func (m *model) SetToolStatus(toolName, status string) {
	if m.toolExecutions == nil {
		m.toolExecutions = make(map[string]string)
	}

	// Update tool status
	m.toolExecutions[toolName] = status

	// Update active tools list
	m.activeTools = nil
	for tool, stat := range m.toolExecutions {
		if stat == "running" {
			m.activeTools = append(m.activeTools, tool)
		}
	}
}

// formatTokenCount formats a token count with K/M suffixes for readability
func formatTokenCount(count int) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	} else if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

// getCurrentWorkingDirectory returns the current working directory with home directory replaced by ~/
func getCurrentWorkingDirectory() string {
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Replace home directory with ~/
	if homeDir, err := os.UserHomeDir(); err == nil && strings.HasPrefix(pwd, homeDir) {
		pwd = "~" + pwd[len(homeDir):]
	}

	return pwd
}

// Update handles messages and updates the component state
func (m *model) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := m.SetSize(msg.Width, msg.Height)
		return m, cmd
	case *runtime.MCPInitStartedEvent:
		m.mcpInit = true
		return m, m.spinner.Init()
	case *runtime.MCPInitFinishedEvent:
		m.mcpInit = false
		return m, nil
	case *runtime.SessionTitleEvent:
		m.sessionTitle = msg.Title
		return m, nil
	case *runtime.AgentInfoEvent:
		m.SetAgentInfo(msg.AgentName, msg.Model, msg.Description)
		return m, nil
	case *runtime.TeamInfoEvent:
		m.SetTeamInfo(msg.AvailableAgents)
		return m, nil
	case *runtime.AgentSwitchingEvent:
		m.SetAgentSwitching(msg.Switching)
		return m, nil
	case *runtime.ToolsetInfoEvent:
		m.SetToolsetInfo(msg.AvailableTools)
		return m, nil
	case *runtime.ToolStatusEvent:
		m.SetToolStatus(msg.ToolName, msg.Status)
		return m, nil
	default:
		if m.working || m.mcpInit {
			var cmd tea.Cmd
			var model layout.Model
			model, cmd = m.spinner.Update(msg)
			m.spinner = model.(spinner.Spinner)
			return m, cmd
		}
		return m, nil
	}
}

// View renders the component
func (m *model) View() string {
	if m.mode == ModeVertical {
		return m.verticalView()
	}

	return m.horizontalView()
}

func (m *model) horizontalView() string {
	pwd := getCurrentWorkingDirectory()

	wi := m.workingIndicator()
	titleGapWidth := m.width - lipgloss.Width(m.sessionTitle) - lipgloss.Width(wi) - 2
	title := fmt.Sprintf("%s%*s%s", m.sessionTitle, titleGapWidth, "", m.workingIndicator())

	gapWidth := m.width - lipgloss.Width(pwd) - lipgloss.Width(m.tokenUsage()) - 2
	return lipgloss.JoinVertical(lipgloss.Top, title, fmt.Sprintf("%s%*s%s", styles.MutedStyle.Render(pwd), gapWidth, "", m.tokenUsage()))
}

func (m *model) verticalView() string {
	topContent := m.sessionTitle + "\n"

	if pwd := getCurrentWorkingDirectory(); pwd != "" {
		topContent += styles.MutedStyle.Render(pwd) + "\n\n"
	}

	topContent += m.tokenUsage()
	topContent += "\n" + m.workingIndicator()

	// Add agent information
	if agentInfo := m.agentInfo(); agentInfo != "" {
		topContent += "\n\n" + agentInfo
	}

	// Add toolset information
	if toolsetInfo := m.toolsetInfo(); toolsetInfo != "" {
		topContent += "\n\n" + toolsetInfo
	}

	m.todoComp.SetSize(m.width)
	todoContent := strings.TrimSuffix(m.todoComp.Render(), "\n")

	// Calculate available height for content
	availableHeight := m.height - 2 // Account for borders
	topHeight := strings.Count(topContent, "\n") + 1
	todoHeight := strings.Count(todoContent, "\n") + 1

	// Calculate padding needed to push todos to bottom
	paddingHeight := max(availableHeight-topHeight-todoHeight, 0)
	for range paddingHeight {
		topContent += "\n"
	}
	topContent += todoContent

	return styles.BaseStyle.
		Width(m.width).
		Height(m.height-2).
		Align(lipgloss.Left, lipgloss.Top).
		Render(topContent)
}

func (m *model) workingIndicator() string {
	if m.mcpInit || m.working {
		label := "Working..."
		if m.mcpInit {
			label = "Initializing MCP servers..."
		}
		indicator := styles.ActiveStyle.Render(m.spinner.View() + " " + label)
		return indicator
	}

	return ""
}

func (m *model) tokenUsage() string {
	totalTokens := m.usage.InputTokens + m.usage.OutputTokens
	var usagePercent float64
	if m.usage.ContextLimit > 0 {
		usagePercent = (float64(m.usage.ContextLength) / float64(m.usage.ContextLimit)) * 100
	}

	percentageText := styles.MutedStyle.Render(fmt.Sprintf("%.0f%%", usagePercent))
	totalTokensText := styles.SubtleStyle.Render(fmt.Sprintf("(%s)", formatTokenCount(totalTokens)))
	costText := styles.MutedStyle.Render(fmt.Sprintf("$%.2f", m.usage.Cost))

	return fmt.Sprintf("%s %s %s", percentageText, totalTokensText, costText)
}

// agentInfo renders the current agent information
func (m *model) agentInfo() string {
	if m.currentAgent == "" {
		return ""
	}

	var content strings.Builder

	// Agent name with highlight and switching indicator
	agentTitle := "AGENT"
	if m.agentSwitching {
		agentTitle = "AGENT ↔" // switching indicator
	}
	content.WriteString(styles.HighlightStyle.Render(agentTitle))
	content.WriteString("\n")

	// Current agent name
	agentName := m.currentAgent
	if m.agentSwitching {
		agentName = "⟳ " + agentName // switching icon
	}
	content.WriteString(styles.MutedStyle.Render(agentName))

	// Team info if multiple agents available
	if len(m.availableAgents) > 1 {
		content.WriteString("\n")
		teamInfo := fmt.Sprintf("Team: %d agents", len(m.availableAgents))
		content.WriteString(styles.SubtleStyle.Render(teamInfo))
	}

	// Model info if available
	if m.agentModel != "" {
		content.WriteString("\n")
		content.WriteString(styles.SubtleStyle.Render("Model: " + m.agentModel))
	}

	// Agent description if available
	if m.agentDescription != "" {
		content.WriteString("\n")

		// Truncate description for sidebar display
		description := m.agentDescription
		maxDescWidth := max(m.width-4, 20) // Leave margin for styling
		if len(description) > maxDescWidth {
			description = description[:maxDescWidth-3] + "..."
		}

		content.WriteString(styles.SubtleStyle.Render(description))
	}

	return content.String()
}

// toolsetInfo renders the current toolset status information
func (m *model) toolsetInfo() string {
	if m.availableTools == 0 {
		return ""
	}

	var content strings.Builder

	// Tools header
	content.WriteString(styles.HighlightStyle.Render("TOOLS"))
	content.WriteString("\n")

	// Available tools count
	content.WriteString(styles.MutedStyle.Render(fmt.Sprintf("%d tools available", m.availableTools)))

	// Active/running tools
	if len(m.activeTools) > 0 {
		content.WriteString("\n")
		for i, tool := range m.activeTools {
			if i > 0 {
				content.WriteString(", ")
			}
			content.WriteString(styles.ActiveStyle.Render("[>] " + tool))
		}
	}

	// Tool execution summary
	runningCount := len(m.activeTools)
	completedCount := 0
	failedCount := 0

	for _, status := range m.toolExecutions {
		switch status {
		case "completed":
			completedCount++
		case "failed":
			failedCount++
		}
	}

	if runningCount > 0 || completedCount > 0 || failedCount > 0 {
		content.WriteString("\n")
		statusParts := []string{}

		if runningCount > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d running", runningCount))
		}
		if completedCount > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d completed", completedCount))
		}
		if failedCount > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d failed", failedCount))
		}

		if len(statusParts) > 0 {
			content.WriteString(styles.SubtleStyle.Render(strings.Join(statusParts, ", ")))
		}
	}

	return content.String()
}

// SetSize sets the dimensions of the component
func (m *model) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	m.todoComp.SetSize(width)
	return nil
}

// GetSize returns the current dimensions
func (m *model) GetSize() (width, height int) {
	return m.width, m.height
}

func (m *model) SetMode(mode Mode) {
	m.mode = mode
}
