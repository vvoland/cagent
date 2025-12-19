package transfertask

import (
	"encoding/json"

	tea "charm.land/bubbletea/v2"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

// Component is a specialized component for rendering transfer_task tool calls.
type Component struct {
	message *types.Message
}

func New(
	msg *types.Message,
	_ *service.SessionState,
) layout.Model {
	return &Component{
		message: msg,
	}
}

func (c *Component) SetSize(int, int) tea.Cmd {
	return nil
}

func (c *Component) Init() tea.Cmd {
	return nil
}

func (c *Component) Update(tea.Msg) (layout.Model, tea.Cmd) {
	return c, nil
}

func (c *Component) View() string {
	var params builtin.TransferTaskArgs
	if err := json.Unmarshal([]byte(c.message.ToolCall.Function.Arguments), &params); err != nil {
		return "" // TODO: Partial tool call
	}

	return styles.AgentBadgeStyle.Render(c.message.Sender) +
		" calls " +
		styles.AgentBadgeStyle.Render(params.Agent) +
		"\n\n" +
		styles.ToolMessageStyle.Render("✓ "+params.Task)
}
