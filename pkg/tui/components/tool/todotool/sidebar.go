package todotool

import (
	"fmt"
	"strings"

	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
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

	var content strings.Builder
	content.WriteString(styles.HighlightStyle.Render("Todo"))
	content.WriteString("\n")

	for _, todo := range c.todos {
		content.WriteString(renderTodoLine(todo, c.width))
		content.WriteString("\n")
	}

	return content.String()
}

func renderTodoLine(todo builtin.Todo, maxWidth int) string {
	icon, iconStyle := renderTodoIcon(todo.Status)
	descStyle := renderTodoDescriptionStyle(todo.Status)

	description := todo.Description
	maxDescWidth := max(maxWidth-2, 3)
	if len(description) > maxDescWidth {
		description = description[:maxDescWidth-3] + "..."
	}

	styledIcon := iconStyle.Render(icon)
	styledDescription := descStyle.Render(description)
	return fmt.Sprintf("%s %s", styledIcon, styledDescription)
}
