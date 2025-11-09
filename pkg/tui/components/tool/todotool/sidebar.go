package todotool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

// SidebarComponent represents the todo display component for the sidebar
type SidebarComponent struct {
	manager *service.TodoManager
	width   int
}

func NewSidebarComponent(manager *service.TodoManager) *SidebarComponent {
	return &SidebarComponent{
		manager: manager,
		width:   20,
	}
}

// SetSize sets the component width
func (c *SidebarComponent) SetSize(width int) {
	c.width = width
}

// SetTodos sets the todo from a tool call, handles create_todo, create_todos, update_todo
func (c *SidebarComponent) SetTodos(toolCall tools.ToolCall) error {
	params, err := parseTodoArgs(toolCall)
	if err != nil {
		return err
	}

	toolName := toolCall.Function.Name
	switch toolName {
	case builtin.ToolNameCreateTodo:
		p := params.(builtin.CreateTodoArgs)
		newID := generateTodoID(c.manager.GetTodos())
		c.manager.AddTodo(newID, p.Description, "pending")

	case builtin.ToolNameCreateTodos:
		p := params.(builtin.CreateTodosArgs)
		for _, desc := range p.Descriptions {
			newID := generateTodoID(c.manager.GetTodos())
			c.manager.AddTodo(newID, desc, "pending")
		}

	case builtin.ToolNameUpdateTodo:
		p := params.(builtin.UpdateTodoArgs)
		c.manager.UpdateTodo(p.ID, p.Status)
	}

	return nil
}

func (c *SidebarComponent) Render() string {
	if len(c.manager.GetTodos()) == 0 {
		return ""
	}

	var content strings.Builder
	content.WriteString(styles.HighlightStyle.Render("TODOs"))
	content.WriteString("\n")

	for _, todo := range c.manager.GetTodos() {
		content.WriteString(renderTodoLine(todo, c.width))
		content.WriteString("\n")
	}

	return content.String()
}

func renderTodoLine(todo types.Todo, maxWidth int) string {
	icon, style := renderTodoIcon(todo.Status)

	description := todo.Description
	maxDescWidth := max(maxWidth-2, 3)
	if len(description) > maxDescWidth {
		description = description[:maxDescWidth-3] + "..."
	}

	styledIcon := style.Render(icon)
	styledDescription := style.Render(description)
	return fmt.Sprintf("%s %s", styledIcon, styledDescription)
}

func parseTodoArgs(toolCall tools.ToolCall) (any, error) {
	toolName := toolCall.Function.Name
	arguments := toolCall.Function.Arguments

	switch toolName {
	case builtin.ToolNameCreateTodo:
		var params builtin.CreateTodoArgs
		if err := json.Unmarshal([]byte(arguments), &params); err != nil {
			return nil, err
		}
		return params, nil
	case builtin.ToolNameCreateTodos:
		var params builtin.CreateTodosArgs
		if err := json.Unmarshal([]byte(arguments), &params); err != nil {
			return nil, err
		}
		return params, nil
	case builtin.ToolNameUpdateTodo:
		var params builtin.UpdateTodoArgs
		if err := json.Unmarshal([]byte(arguments), &params); err != nil {
			return nil, err
		}
		return params, nil
	case builtin.ToolNameListTodos:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown tool name: %s", toolName)
	}
}

func generateTodoID(todos []types.Todo) string {
	return fmt.Sprintf("todo_%d", len(todos)+1)
}
