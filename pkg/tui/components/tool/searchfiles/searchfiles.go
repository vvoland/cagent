package searchfiles

import (
	"fmt"
	"strings"

	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/types"
)

func New(msg *types.Message, sessionState *service.SessionState) layout.Model {
	return toolcommon.NewBase(msg, sessionState, toolcommon.SimpleRendererWithResult(
		toolcommon.ExtractField(func(a builtin.SearchFilesArgs) string { return a.Pattern }),
		extractResult,
	))
}

func extractResult(msg *types.Message) string {
	content := msg.Content
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
			return strings.TrimSuffix(firstLine, ":")
		}
	}

	// Fallback
	return content
}
