package transfertask

import (
	"encoding/json"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

// Component is a specialized component for rendering transfer_task tool calls.
type Component struct {
	message  *types.Message
	renderer *glamour.TermRenderer
}

func New(
	msg *types.Message,
	_ *app.App,
	renderer *glamour.TermRenderer,
	_ *service.SessionState,
) layout.Model {
	return &Component{
		message:  msg,
		renderer: renderer,
	}
}

// SetSize implements [layout.Model].
func (c *Component) SetSize(width, height int) tea.Cmd {
	return nil
}

// Init implements [layout.Model].
func (c *Component) Init() tea.Cmd {
	return nil
}

// Update implements [layout.Model].
func (c *Component) Update(tea.Msg) (layout.Model, tea.Cmd) {
	return c, nil
}

// View implements [layout.Model].
func (c *Component) View() string {
	var params builtin.TransferTaskArgs
	if err := json.Unmarshal([]byte(c.message.ToolCall.Function.Arguments), &params); err != nil {
		return "" // TODO: Partial tool call
	}

	return c.message.Sender + " -> " + params.Agent + ": " + styles.MutedStyle.Render(params.Task)
}
