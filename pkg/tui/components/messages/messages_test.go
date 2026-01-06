package messages

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/types"
)

func TestViewDoesNotWrapWideLines(t *testing.T) {
	t.Parallel()

	sessionState := &service.SessionState{}
	m := NewScrollableView(20, 5, sessionState).(*model)
	m.SetSize(20, 5)

	msg := types.Agent(types.MessageTypeAssistant, "", strings.Repeat("x", 200))
	m.messages = append(m.messages, msg)
	m.views = append(m.views, m.createMessageView(msg))

	out := m.View()
	for _, line := range strings.Split(out, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), 20)
	}
}
