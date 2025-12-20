package readfile

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/docker/cagent/pkg/paths"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/types"
)

// Component is a specialized component for rendering read_file tool calls.
type Component struct {
	message *types.Message
	spinner spinner.Spinner
	width   int
	height  int
}

// New creates a new read file component.
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
		model, cmd := c.spinner.Update(msg)
		c.spinner = model.(spinner.Spinner)
		return c, cmd
	}

	return c, nil
}

func (c *Component) View() string {
	msg := c.message

	var args builtin.ReadFileArgs
	if err := json.Unmarshal([]byte(msg.ToolCall.Function.Arguments), &args); err != nil {
		return toolcommon.RenderTool(msg, c.spinner, "", "", c.width)
	}

	var details string
	if msg.ToolResult != nil && msg.ToolResult.Meta != nil {
		meta := msg.ToolResult.Meta.(builtin.ReadFileMeta)
		if meta.Error != "" {
			details = meta.Error
		} else {
			details = fmt.Sprintf("%d lines", meta.LineCount)
		}
	}

	return toolcommon.RenderTool(msg, c.spinner, shortenPath(args.Path), details, c.width)
}

// shortenPath replaces home directory with ~ and shortens paths
func shortenPath(path string) string {
	homeDir := paths.GetHomeDir()
	if homeDir != "" && strings.HasPrefix(path, homeDir) {
		return "~" + strings.TrimPrefix(path, homeDir)
	}
	return path
}
