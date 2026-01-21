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

// New creates the edit_file tool UI model.
func New(msg *types.Message, sessionState *service.SessionState) layout.Model {
	return toolcommon.NewBase(msg, sessionState, render)
}

// render displays the edit_file tool output in the TUI.
// It prioritizes the agent-provided friendly header when available,
// hides results when collapsed by the user, and skips diff rendering
// on tool execution errors to avoid layout issues.
func render(
	msg *types.Message,
	s spinner.Spinner,
	sessionState *service.SessionState,
	width,
	_ int,
) string {
	// Parse tool arguments to extract the file path for display.
	var args builtin.EditFileArgs
	if err := json.Unmarshal([]byte(msg.ToolCall.Function.Arguments), &args); err != nil {
		// If arguments cannot be parsed, fail silently to avoid breaking the TUI.
		return ""
	}

	// Check for friendly description first
	var content string
	if header, ok := toolcommon.RenderFriendlyHeader(msg, s); ok {
		content = header
	} else {
		content = fmt.Sprintf(
			"%s%s %s",
			toolcommon.Icon(msg, s),
			styles.ToolName.Render(msg.ToolDefinition.DisplayName()),
			styles.ToolMessageStyle.Render(toolcommon.ShortenPath(args.Path)),
		)
	}

	// Tool results are hidden when the user collapses them.
	if sessionState.HideToolResults() {
		return content
	}

	// When the edit fails, do not render a diff.
	// Instead, render the error/rejection as a single-line tool message.
	// Rendering diffs on failed edits can break layout and scroll calculations.
	if msg.ToolStatus == types.ToolStatusError {
		if msg.Content != "" {
			// Render error/rejection as a single-line tool error message
			// with consistent spacing and styling.
			errorText := fmt.Sprintf("  %s", msg.Content)

			// Truncate to available width to avoid wrapping
			errorLine := styles.ToolErrorMessageStyle.
				MaxWidth(width).
				Render(errorText)

			content += "\n" + errorLine
		}
		return content
	}

	// Successful (or pending/confirmation) execution:
	// render the diff output inside the ToolCallResult container.
	if msg.ToolCall.Function.Arguments != "" {
		// Calculate available width for diff rendering, accounting for
		// ToolCallResult frame padding.
		contentWidth := width - styles.ToolCallResult.GetHorizontalFrameSize()

		content += "\n" + styles.ToolCallResult.Render(
			renderEditFile(
				msg.ToolCall,
				contentWidth,
				sessionState.SplitDiffView(),
				msg.ToolStatus,
			),
		)
	}

	return content
}
