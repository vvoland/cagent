package sidebar

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"github.com/docker/cagent/internal/tui/components/todo"
	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/util"
	"github.com/docker/cagent/pkg/runtime"
)

// Model represents a sidebar component
type Model interface {
	util.Model
	layout.Sizeable

	SetTitle(title string)
	SetTokenUsage(usage *runtime.Usage)
	SetTodoArguments(toolName, arguments string) error
}

// model implements Model
type model struct {
	width    int
	height   int
	title    string
	usage    *runtime.Usage
	todoComp *todo.Component
}

// New creates a new sidebar component
func New() Model {
	return &model{
		width:    20, // Default width
		height:   24, // Default height
		usage:    &runtime.Usage{},
		todoComp: todo.NewComponent(),
		title:    "New Session",
	}
}

// Init initializes the component
func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) SetTitle(title string) {
	m.title = title
}

func (m *model) SetTokenUsage(usage *runtime.Usage) {
	m.usage = usage
}

func (m *model) SetTodoArguments(toolName, arguments string) error {
	if toolName == "todo_write" {
		return m.todoComp.ParseTodoWriteArguments(arguments)
	}
	return m.todoComp.ParseTodoArguments(toolName, arguments)
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
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		cmd := m.SetSize(ws.Width, ws.Height)
		return m, cmd
	}
	return m, nil
}

// View renders the component
func (m *model) View() string {
	// Calculate token usage metrics
	totalTokens := m.usage.InputTokens + m.usage.OutputTokens
	var usagePercent float64
	if m.usage.ContextLimit > 0 {
		usagePercent = (float64(m.usage.ContextLength) / float64(m.usage.ContextLimit)) * 100
	}

	// Define color styles for the usage display
	greyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
	lightGreyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C0C0C0"))

	// Build top content (title + pwd + token usage)
	topContent := m.title + "\n\n"

	// Add current working directory in grey
	if pwd := getCurrentWorkingDirectory(); pwd != "" {
		topContent += greyStyle.Render(pwd) + "\n\n"
	}

	// Format each part with its respective color
	percentageText := greyStyle.Render(fmt.Sprintf("%.0f%%", usagePercent))
	totalTokensText := lightGreyStyle.Render(fmt.Sprintf("(%s)", formatTokenCount(totalTokens)))
	costText := greyStyle.Render(fmt.Sprintf("$%.2f", m.usage.Cost))

	topContent += fmt.Sprintf("%s %s %s", percentageText, totalTokensText, costText)

	// Get todo content (if any)
	m.todoComp.SetSize(m.width)
	todoContent := m.todoComp.Render()

	// If we have todos, create a layout with todos at the bottom
	if todoContent != "" {
		// Remove trailing newline from todoContent if present
		todoContent = strings.TrimSuffix(todoContent, "\n")

		// Calculate available height for content
		availableHeight := m.height - 2 // Account for borders
		topHeight := strings.Count(topContent, "\n") + 1
		todoHeight := strings.Count(todoContent, "\n") + 1

		// Calculate padding needed to push todos to bottom
		paddingHeight := availableHeight - topHeight - todoHeight
		if paddingHeight < 0 {
			paddingHeight = 0
		}

		// Build final content with padding
		finalContent := topContent
		for range paddingHeight {
			finalContent += "\n"
		}
		finalContent += todoContent

		sidebarStyle := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-2).
			Align(lipgloss.Left, lipgloss.Top)

		return sidebarStyle.Render(finalContent)
	} else {
		// No todos, just render top content normally
		sidebarStyle := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-2).
			Align(lipgloss.Left, lipgloss.Top)

		return sidebarStyle.Render(topContent)
	}
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
