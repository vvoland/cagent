package writefile

import (
	"github.com/docker/docker-agent/pkg/tools/builtin"
	"github.com/docker/docker-agent/pkg/tui/components/toolcommon"
	"github.com/docker/docker-agent/pkg/tui/core/layout"
	"github.com/docker/docker-agent/pkg/tui/service"
	"github.com/docker/docker-agent/pkg/tui/types"
)

func New(msg *types.Message, sessionState service.SessionStateReader) layout.Model {
	return toolcommon.NewBase(msg, sessionState, toolcommon.SimpleRenderer(
		toolcommon.ExtractField(func(a builtin.WriteFileArgs) string { return a.Path }),
	))
}
