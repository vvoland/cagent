package readmultiplefiles

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
		model, cmd := c.spinner.Update(msg)
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
		return toolcommon.RenderTool(msg, c.spinner, "", "", c.width)
	}

	// For pending/running state, show files being read
	if msg.ToolStatus == types.ToolStatusPending || msg.ToolStatus == types.ToolStatusRunning {
		params := formatFilesList(args.Paths)
		return toolcommon.RenderTool(msg, c.spinner, params, "", c.width)
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

		// Icon / Tool name / File path
		nameStyle := styles.ToolName
		icon := toolcommon.Icon(msg, c.spinner)
		if summary.isError {
			nameStyle = styles.ToolNameError
			icon = toolcommon.Icon(&types.Message{ToolStatus: types.ToolStatusError}, c.spinner)
		}
		readCall := icon + nameStyle.Render("Read")
		if summary.path != "" {
			readCall += " " + summary.path
		}

		// Output aligned to the right
		outputStyle := styles.ToolMessageStyle
		if summary.isError {
			outputStyle = styles.ToolErrorMessageStyle
		}
		output := outputStyle.Render(summary.output)
		padding := c.width - lipgloss.Width(readCall) - lipgloss.Width(output) - 2
		if padding > 0 {
			output = strings.Repeat(" ", padding) + output
		}

		content.WriteString(readCall)
		content.WriteString(" ")
		content.WriteString(output)
	}

	// Apply tool message styling
	return styles.RenderComposite(styles.ToolMessageStyle.Width(c.width-1), content.String())
}

type fileSummary struct {
	path    string
	output  string
	isError bool
}

// formatSummaryLines creates a summary for each file from metadata
func formatSummaryLines(meta *builtin.ReadMultipleFilesMeta) []fileSummary {
	if meta == nil || len(meta.Files) == 0 {
		return nil
	}

	homeDir := paths.GetHomeDir()
	var summaries []fileSummary

	for _, file := range meta.Files {
		path := shortenPath(file.Path, homeDir)
		var output string
		if file.Error != "" {
			output = " " + file.Error
		} else {
			output = fmt.Sprintf(" %d lines", file.LineCount)
		}

		summaries = append(summaries, fileSummary{
			path:    path,
			output:  output,
			isError: file.Error != "",
		})
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
