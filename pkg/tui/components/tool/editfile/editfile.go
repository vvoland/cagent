package editfile

import (
	"encoding/json"
	"fmt"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

type ToggleDiffViewMsg struct{}

func New(msg *types.Message, sessionState *service.SessionState) layout.Model {
	return toolcommon.NewBase(msg, sessionState, makeRenderer(sessionState))
}

func makeRenderer(sessionState *service.SessionState) toolcommon.Renderer {
	return func(msg *types.Message, s spinner.Spinner, width, _ int) string {
		var args builtin.EditFileArgs
		if err := json.Unmarshal([]byte(msg.ToolCall.Function.Arguments), &args); err != nil {
			return ""
		}

		displayName := msg.ToolDefinition.DisplayName()
		content := fmt.Sprintf("%s%s %s",
			toolcommon.Icon(msg, s),
			styles.ToolName.Render(displayName),
			styles.ToolMessageStyle.Render(toolcommon.ShortenPath(args.Path)))

		if msg.ToolCall.Function.Arguments != "" {
			content += "\n" + styles.ToolCallResult.Render(
				renderEditFile(msg.ToolCall, width-1, sessionState.SplitDiffView, msg.ToolStatus))
		}

		if (msg.ToolStatus == types.ToolStatusCompleted || msg.ToolStatus == types.ToolStatusError) && msg.Content != "" {
			content += toolcommon.FormatToolResult(msg.Content, width)
		}

		return content
	}
}
