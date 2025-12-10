package searchfiles

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/types"
)

// Component is a specialized component for rendering search_files tool calls.
type Component struct {
	message *types.Message
	spinner spinner.Spinner
	width   int
	height  int
}

// New creates a new search files component.
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

	// Parse arguments
	var args builtin.SearchFilesArgs
	if err := json.Unmarshal([]byte(msg.ToolCall.Function.Arguments), &args); err != nil {
		return toolcommon.RenderTool(toolcommon.Icon(msg.ToolStatus), msg.ToolDefinition.DisplayName(), c.spinner.View(), "", c.width)
	}

	// Format display name with pattern
	displayName := fmt.Sprintf("%s(%q)", msg.ToolDefinition.DisplayName(), args.Pattern)

	// For pending/running state, show spinner
	if msg.ToolStatus == types.ToolStatusPending || msg.ToolStatus == types.ToolStatusRunning {
		return toolcommon.RenderTool(toolcommon.Icon(msg.ToolStatus), displayName, c.spinner.View(), "", c.width)
	}

	// For completed/error state, show concise summary
	summary := formatSummary(msg.Content)
	params := fmt.Sprintf(": %s", summary)

	return toolcommon.RenderTool(toolcommon.Icon(msg.ToolStatus), displayName, params, "", c.width)
}

// formatSummary creates a concise summary of the search results
func formatSummary(content string) string {
	if content == "" {
		return "no result"
	}

	// Handle "No files found" case
	if strings.HasPrefix(content, "No files found") {
		return "no file found"
	}

	// Handle error cases
	if strings.HasPrefix(content, "Error") {
		return content
	}

	// Parse "X files found:\n..." format
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		firstLine := lines[0]
		// Extract count from "X files found:" or "X file found:"
		if strings.Contains(firstLine, "file found:") || strings.Contains(firstLine, "files found:") {
			// Check if it's a single file
			if strings.HasPrefix(firstLine, "1 file found:") && len(lines) > 1 {
				// Return "one file found: filename"
				fileName := strings.TrimSpace(lines[1])
				return fmt.Sprintf("one file found: %s", fileName)
			}
			// Multiple files
			return strings.TrimSuffix(firstLine, ":") + "."
		}
	}

	// Fallback
	return content
}
