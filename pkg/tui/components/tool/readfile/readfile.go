package readfile

import (
	"encoding/json"
	"fmt"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/types"
)

func New(msg *types.Message, sessionState *service.SessionState) layout.Model {
	return toolcommon.NewBase(msg, sessionState, toolcommon.SimpleRendererWithResult(extractPath, extractResult))
}

func extractPath(args string) string {
	var a builtin.ReadFileArgs
	if err := json.Unmarshal([]byte(args), &a); err != nil {
		return ""
	}
	return toolcommon.ShortenPath(a.Path)
}

func extractResult(msg *types.Message) string {
	if msg.ToolResult == nil || msg.ToolResult.Meta == nil {
		return ""
	}
	meta, ok := msg.ToolResult.Meta.(builtin.ReadFileMeta)
	if !ok {
		return ""
	}
	if meta.Error != "" {
		return meta.Error
	}
	return fmt.Sprintf("%d lines", meta.LineCount)
}
