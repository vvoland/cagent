package readmultiplefiles

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
	"github.com/docker/cagent/pkg/tui/styles"
	"github.com/docker/cagent/pkg/tui/types"
)

// Component is a specialized component for rendering read_multiple_files tool calls.
type Component struct {
	message *types.Message
	spinner spinner.Spinner
	width   int
	height  int
}

// New creates a new read multiple files component.
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
	var args builtin.ReadMultipleFilesArgs
	if err := json.Unmarshal([]byte(msg.ToolCall.Function.Arguments), &args); err != nil {
		return toolcommon.RenderTool(msg, c.spinner, "Read Multiple Files", "", c.width)
	}

	// For pending/running state, show files being read
	if msg.ToolStatus == types.ToolStatusPending || msg.ToolStatus == types.ToolStatusRunning {
		params := formatFilesList(args.Paths)
		return toolcommon.RenderTool(msg, c.spinner, "Read Multiple Files "+params, "", c.width)
	}

	// For completed/error state, render header line followed by each file line
	var meta *builtin.ReadMultipleFilesMeta
	if msg.ToolResult != nil {
		if m, ok := msg.ToolResult.Meta.(builtin.ReadMultipleFilesMeta); ok {
			meta = &m
		}
	}

	// Each file on its own line with checkmark
	var content strings.Builder
	for _, summary := range formatSummaryLines(meta) {
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		fmt.Fprintf(&content, "%s %s %s", toolcommon.Icon(msg, c.spinner), summary.displayName, summary.params)
	}

	// Apply tool message styling
	return styles.RenderComposite(styles.ToolMessageStyle.Width(c.width-1), content.String())
}

type fileSummary struct {
	displayName string
	params      string
}

// formatSummaryLines creates a summary for each file from metadata
func formatSummaryLines(meta *builtin.ReadMultipleFilesMeta) []fileSummary {
	if meta == nil || len(meta.Files) == 0 {
		return nil
	}

	homeDir := paths.GetHomeDir()
	summaries := make([]fileSummary, 0, len(meta.Files))

	for _, file := range meta.Files {
		params := shortenPath(file.Path, homeDir)
		if file.Error != "" {
			params += " " + file.Error
		} else {
			params += fmt.Sprintf(" %d lines", file.LineCount)
		}
		summaries = append(summaries, fileSummary{displayName: "Read", params: params})
	}

	return summaries
}

// formatFilesList formats a list of file paths for display
func formatFilesList(filePaths []string) string {
	if len(filePaths) == 0 {
		return ""
	}

	// Shorten paths by using home directory symbol
	homeDir := paths.GetHomeDir()
	shortened := make([]string, len(filePaths))
	for i, p := range filePaths {
		shortened[i] = shortenPath(p, homeDir)
	}

	if len(shortened) == 1 {
		return shortened[0]
	}

	// For multiple files, show comma-separated list
	return strings.Join(shortened, ", ")
}

// shortenPath replaces home directory with ~ and shortens paths
func shortenPath(path, homeDir string) string {
	if homeDir != "" && strings.HasPrefix(path, homeDir) {
		return "~" + strings.TrimPrefix(path, homeDir)
	}
	return path
}
