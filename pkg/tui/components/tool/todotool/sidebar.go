package todotool

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/tab"
	"github.com/docker/cagent/pkg/tui/styles"
)

// SidebarComponent represents the todo display component for the sidebar
type SidebarComponent struct {
	todos []builtin.Todo
	width int
}

func NewSidebarComponent() *SidebarComponent {
	return &SidebarComponent{
		width: 20,
	}
}

func (c *SidebarComponent) SetSize(width int) {
	c.width = width
}

func (c *SidebarComponent) SetTodos(result *tools.ToolCallResult) error {
	if result == nil || result.Meta == nil {
		return nil
	}

	todos, ok := result.Meta.([]builtin.Todo)
	if !ok {
		return nil
	}

	c.todos = todos
	return nil
}

func (c *SidebarComponent) Render() string {
	if len(c.todos) == 0 {
		return ""
	}

	var lines []string
	for _, todo := range c.todos {
		lines = append(lines, c.renderTodoLine(todo))
	}

	return c.renderTab("TO-DO", strings.Join(lines, "\n"))
}

func (c *SidebarComponent) renderTodoLine(todo builtin.Todo) string {
	icon, style := renderTodoIcon(todo.Status)

	description := todo.Description
	maxDescWidth := c.width - 4
	if lipgloss.Width(description) > maxDescWidth {
		// Truncate by runes to handle Unicode properly
		runes := []rune(description)
		for lipgloss.Width(string(runes)) > maxDescWidth-1 && len(runes) > 0 {
			runes = runes[:len(runes)-1]
		}
		description = string(runes) + "…"
	}

	return styles.TabPrimaryStyle.Render(style.Render(icon + " " + description))
}

func (c *SidebarComponent) renderTab(title, content string) string {
	return tab.Render(title, content, c.width-2)
}
