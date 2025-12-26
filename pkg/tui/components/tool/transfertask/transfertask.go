package transfertask

import (
	"encoding/json"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

func New(msg *types.Message, sessionState *service.SessionState) layout.Model {
	return toolcommon.NewBase(msg, sessionState, render)
}

func render(msg *types.Message, _ spinner.Spinner, _, _ int) string {
	var params builtin.TransferTaskArgs
	if err := json.Unmarshal([]byte(msg.ToolCall.Function.Arguments), &params); err != nil {
		return ""
	}

	return styles.AgentBadgeStyle.Render(msg.Sender) +
		" calls " +
		styles.AgentBadgeStyle.Render(params.Agent) +
		"\n\n" +
		styles.ToolMessageStyle.Render(styles.ToolCompletedIcon.Render("✓")+" "+params.Task)
}
