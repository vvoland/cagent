package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/history"
	"github.com/docker/cagent/pkg/tui/components/editor"
)

func TestProcessMessage_AddsToHistory(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	hist, err := history.New(history.WithBaseDir(homeDir))
	require.NoError(t, err)

	p := &chatPage{
		history: hist,
	}

	p.processMessage(editor.SendMsg{Content: "/new"})
	p.processMessage(editor.SendMsg{Content: "/compact"})
	p.processMessage(editor.SendMsg{Content: "/copy"})

	assert.Equal(t, []string{"/new", "/compact", "/copy"}, hist.Messages)

	hist2, err := history.New(history.WithBaseDir(homeDir))
	require.NoError(t, err)
	assert.Equal(t, []string{"/new", "/compact", "/copy"}, hist2.Messages)
}
