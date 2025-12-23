package shell

import (
	"encoding/json"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/types"
)

func New(msg *types.Message, sessionState *service.SessionState) layout.Model {
	return toolcommon.NewBase(msg, sessionState, toolcommon.SimpleRenderer(extractCmd))
}

func extractCmd(args string) string {
	var a builtin.RunShellArgs
	if err := json.Unmarshal([]byte(args), &a); err != nil {
		return ""
	}
	return a.Cmd
}
