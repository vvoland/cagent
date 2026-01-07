package chat

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/docker/cagent/pkg/tui/components/notification"
)

// editorDoneMsg is sent when the external editor finishes to trigger a TUI refresh.
type editorDoneMsg struct{}

// openExternalEditor opens the current editor content in an external editor.
// It suspends the TUI, runs the editor, and returns the result.
func (p *chatPage) openExternalEditor() tea.Cmd {
	content := p.editor.Value()

	// Create a temporary file with the current content
	tmpFile, err := os.CreateTemp("", "cagent-*.md")
	if err != nil {
		return notification.ErrorCmd(fmt.Sprintf("Failed to create temp file: %v", err))
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return notification.ErrorCmd(fmt.Sprintf("Failed to write temp file: %v", err))
	}
	tmpFile.Close()

	// Get the editor command (VISUAL, EDITOR, or platform default)
	editorCmd := cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR"))
	if editorCmd == "" {
		if runtime.GOOS == "windows" {
			editorCmd = "notepad"
		} else {
			editorCmd = "vi"
		}
	}

	// Parse editor command (may include arguments like "code --wait")
	parts := strings.Fields(editorCmd)
	args := append(parts[1:], tmpPath)
	cmd := exec.Command(parts[0], args...)

	// Use tea.ExecProcess to properly suspend the TUI and run the editor
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			os.Remove(tmpPath)
			return notification.ErrorCmd(fmt.Sprintf("Editor error: %v", err))
		}

		updatedContent, readErr := os.ReadFile(tmpPath)
		os.Remove(tmpPath)

		if readErr != nil {
			return notification.ErrorCmd(fmt.Sprintf("Failed to read edited file: %v", readErr))
		}

		// Trim trailing newline that editors often add
		content := strings.TrimSuffix(string(updatedContent), "\n")

		// If content is empty, just clear the editor
		if strings.TrimSpace(content) == "" {
			p.editor.SetValue("")
			return editorDoneMsg{}
		}

		// Clear the editor and automatically submit the content
		p.editor.SetValue(content)
		return editorDoneMsg{}
	})
}
