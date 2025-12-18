package todotool

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

// Component represents a unified todo tool component that handles all todo operations.
// It determines which operation to display based on the tool call name.
type Component struct {
	message *types.Message
	spinner spinner.Spinner
	width   int
	height  int
}

// New creates a new unified todo component.
// This component handles create, create_multiple, list, and update operations.
func New(
	msg *types.Message,
	_ *service.SessionState,
) layout.Model {
	return &Component{
		message: msg,
		spinner: spinner.New(spinner.ModeSpinnerOnly),
		width:   80,
		height:  1,
	}
}

func (c *Component) SetSize(width, height int) tea.Cmd {
	c.width = width
	c.height = height
	return nil
}

func (c *Component) Init() tea.Cmd {
	if c.message.ToolStatus == types.ToolStatusPending || c.message.ToolStatus == types.ToolStatusRunning {
		return c.spinner.Init()
	}
	return nil
}

func (c *Component) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	if c.message.ToolStatus == types.ToolStatusPending || c.message.ToolStatus == types.ToolStatusRunning {
		var cmd tea.Cmd
		var model layout.Model
		model, cmd = c.spinner.Update(msg)
		c.spinner = model.(spinner.Spinner)
		return c, cmd
	}

	return c, nil
}

func (c *Component) View() string {
	msg := c.message
	toolName := msg.ToolCall.Function.Name

	// Render based on tool type
	switch toolName {
	case builtin.ToolNameCreateTodo, builtin.ToolNameCreateTodos:
		return c.renderTodos()
	case builtin.ToolNameUpdateTodo:
		return "" // We've got todos in the sidebar
	case builtin.ToolNameListTodos:
		return c.renderList()
	default:
		panic("Unsupported todo tool: " + toolName)
	}
}

func (c *Component) renderTodos() string {
	msg := c.message
	displayName := msg.ToolDefinition.DisplayName()

	var content strings.Builder
	fmt.Fprintf(&content, "%s %s", toolcommon.Icon(msg, c.spinner), styles.ToolMessageStyle.Render(displayName))

	return styles.RenderComposite(styles.ToolMessageStyle.Width(c.width-1), content.String())
}

func (c *Component) renderList() string {
	msg := c.message
	displayName := msg.ToolDefinition.DisplayName()
	var content strings.Builder
	content.WriteString(fmt.Sprintf("%s %s", toolcommon.Icon(msg, c.spinner), styles.ToolMessageStyle.Render(displayName)))

	if msg.ToolResult != nil && msg.ToolResult.Meta != nil {
		if todos, ok := msg.ToolResult.Meta.([]builtin.Todo); ok {
			for _, todo := range todos {
				icon, style := renderTodoIcon(todo.Status)
				todoLine := fmt.Sprintf("\n%s %s", style.Render(icon), style.Render(todo.Description))
				content.WriteString(todoLine)
			}
		}
	}

	return styles.RenderComposite(styles.ToolMessageStyle.Width(c.width-1), content.String())
}
