package api

import (
	"encoding/json"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/docker/go-units"

	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

type Component struct {
	message *types.Message
	spinner spinner.Spinner
	width   int
	height  int
}

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

	var args map[string]any
	if err := json.Unmarshal([]byte(msg.ToolCall.Function.Arguments), &args); err != nil {
		return toolcommon.RenderTool(msg, c.spinner, msg.ToolDefinition.DisplayName(), "", c.width)
	}

	// Build the display name with inline result
	displayName := msg.ToolDefinition.DisplayName()

	// Extract argument summary for the tool call display
	var params string
	if argsText := formatArgs(args); argsText != "" {
		params = "(" + argsText + ")"
	}

	// Add inline result/progress after the tool name
	switch msg.ToolStatus {
	case types.ToolStatusRunning:
		// While running, show what we're calling
		endpoint := extractEndpoint(args)
		if endpoint != "" {
			params += styles.MutedStyle.Render(": Calling " + endpoint)
		}
	case types.ToolStatusCompleted:
		// When completed, show a brief summary inline
		resultSummary := extractSummary(msg.Content)
		params += styles.MutedStyle.Render(": " + resultSummary)
	}

	// Render everything on one line
	return toolcommon.RenderTool(msg, c.spinner, displayName+" "+params, "", c.width)
}

// extractEndpoint tries to find the endpoint/URL being called
func extractEndpoint(args map[string]any) string {
	if endpoint, ok := args["endpoint"].(string); ok {
		return endpoint
	}
	if url, ok := args["url"].(string); ok {
		return url
	}
	return ""
}

// formatArgs creates a concise string representation of the arguments
func formatArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}

	// Check for URL or URLs field (common in fetch tools)
	if urlVal, ok := args["url"].(string); ok && urlVal != "" {
		return urlVal
	}
	if urlsVal, ok := args["urls"].([]any); ok && len(urlsVal) > 0 {
		// Extract just the URLs from the array
		var urls []string
		for _, u := range urlsVal {
			if urlStr, ok := u.(string); ok {
				urls = append(urls, urlStr)
			}
		}
		if len(urls) == 1 {
			return urls[0]
		} else if len(urls) > 1 {
			return fmt.Sprintf("%s (+%d more)", urls[0], len(urls)-1)
		}
	}

	// Try to find common parameter names that might indicate what's being queried
	for _, key := range []string{"query", "q", "search", "message", "prompt", "text"} {
		if val, ok := args[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}

	// Fallback: show JSON
	b, _ := json.Marshal(args)
	return string(b)
}

// extractSummary tries to extract a meaningful summary from the API response
func extractSummary(content string) string {
	return fmt.Sprintf("Received %s", units.HumanSize(float64(len(content))))
}
