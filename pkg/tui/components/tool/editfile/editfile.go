package editfile

import (
	"encoding/json"
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

type ToggleDiffViewMsg struct{}

// Component is a specialized component for rendering edit_file tool calls.
type Component struct {
	message      *types.Message
	renderer     *glamour.TermRenderer
	spinner      spinner.Model
	width        int
	height       int
	sessionState *service.SessionState
}

func New(
	msg *types.Message,
	renderer *glamour.TermRenderer,
	sessionState *service.SessionState,
) layout.Model {
	return &Component{
		message:      msg,
		renderer:     renderer,
		spinner:      spinner.New(spinner.WithSpinner(spinner.Points)),
		width:        80,
		height:       1,
		sessionState: sessionState,
	}
}

// SetSize implements [layout.Model].
func (c *Component) SetSize(width, height int) tea.Cmd {
	c.width = width
	c.height = height
	return nil
}

// Init implements [layout.Model].
func (c *Component) Init() tea.Cmd {
	if c.message.ToolStatus == types.ToolStatusPending || c.message.ToolStatus == types.ToolStatusRunning {
		return c.spinner.Tick
	}
	return nil
}

// Update implements [layout.Model].
func (c *Component) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	// if _, ok := msg.(ToggleDiffViewMsg); ok {
	// 	c.sessionState.ToggleSplitDiffView()
	// 	return c, nil
	// }

	// Handle spinner updates
	if c.message.ToolStatus == types.ToolStatusPending || c.message.ToolStatus == types.ToolStatusRunning {
		var cmd tea.Cmd
		c.spinner, cmd = c.spinner.Update(msg)
		return c, cmd
	}

	return c, nil
}

// View implements [layout.Model].
func (c *Component) View() string {
	msg := c.message
	var args builtin.EditFileArgs
	if err := json.Unmarshal([]byte(msg.ToolCall.Function.Arguments), &args); err != nil {
		return ""
	}

	displayName := msg.ToolDefinition.DisplayName()
	content := fmt.Sprintf("%s %s %s", toolcommon.Icon(msg.ToolStatus), styles.HighlightStyle.Render(displayName), styles.MutedStyle.Render(args.Path))

	if msg.ToolStatus == types.ToolStatusPending || msg.ToolStatus == types.ToolStatusRunning {
		content += " " + c.spinner.View()
	}

	if msg.ToolCall.Function.Arguments != "" {
		content += "\n\n" + styles.ToolCallResult.Render(renderEditFile(msg.ToolCall, c.width-4, c.sessionState.SplitDiffView))
	}

	var resultContent string
	if (msg.ToolStatus == types.ToolStatusCompleted || msg.ToolStatus == types.ToolStatusError) && msg.Content != "" {
		resultContent = toolcommon.FormatToolResult(msg.Content, c.width)
	}

	return styles.BaseStyle.PaddingLeft(2).PaddingTop(1).Render(content + resultContent)
}
