package messages

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/tools"
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

func TestLoadFromSessionIncludesReasoningContent(t *testing.T) {
	t.Parallel()

	sessionState := &service.SessionState{}
	m := NewScrollableView(80, 24, sessionState).(*model)
	m.SetSize(80, 24)

	sess := &session.Session{
		ID: "test-session",
		Messages: []session.Item{
			session.NewMessageItem(&session.Message{
				Message: chat.Message{
					Role:    chat.MessageRoleUser,
					Content: "Hello",
				},
			}),
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:             chat.MessageRoleAssistant,
					ReasoningContent: "Let me think about this...",
					Content:          "Hello back!",
				},
			}),
		},
	}

	m.LoadFromSession(sess)

	require.Len(t, m.messages, 3)
	// User message first
	assert.Equal(t, types.MessageTypeUser, m.messages[0].Type)
	assert.Equal(t, "Hello", m.messages[0].Content)
	// Reasoning content second
	assert.Equal(t, types.MessageTypeAssistantReasoning, m.messages[1].Type)
	assert.Equal(t, "Let me think about this...", m.messages[1].Content)
	assert.Equal(t, "root", m.messages[1].Sender)
	// Assistant content third
	assert.Equal(t, types.MessageTypeAssistant, m.messages[2].Type)
	assert.Equal(t, "Hello back!", m.messages[2].Content)
	assert.Equal(t, "root", m.messages[2].Sender)
}

func TestLoadFromSessionReasoningOrderWithToolCalls(t *testing.T) {
	t.Parallel()

	sessionState := &service.SessionState{}
	m := NewScrollableView(80, 24, sessionState).(*model)
	m.SetSize(80, 24)

	sess := &session.Session{
		ID: "test-session",
		Messages: []session.Item{
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:             chat.MessageRoleAssistant,
					ReasoningContent: "I should call a tool...",
					ToolCalls: []tools.ToolCall{
						{ID: "call-1", Function: tools.FunctionCall{Name: "test_tool", Arguments: "{}"}},
					},
					ToolDefinitions: []tools.Tool{
						{Name: "test_tool", Description: "A test tool"},
					},
					Content: "Tool result processed.",
				},
			}),
		},
	}

	m.LoadFromSession(sess)

	require.Len(t, m.messages, 3)
	// Reasoning comes first
	assert.Equal(t, types.MessageTypeAssistantReasoning, m.messages[0].Type)
	assert.Equal(t, "I should call a tool...", m.messages[0].Content)
	// Tool call comes second
	assert.Equal(t, types.MessageTypeToolCall, m.messages[1].Type)
	assert.Equal(t, "test_tool", m.messages[1].ToolCall.Function.Name)
	// Assistant content comes last
	assert.Equal(t, types.MessageTypeAssistant, m.messages[2].Type)
	assert.Equal(t, "Tool result processed.", m.messages[2].Content)
}

func TestLoadFromSessionReasoningOnlyNoContent(t *testing.T) {
	t.Parallel()

	sessionState := &service.SessionState{}
	m := NewScrollableView(80, 24, sessionState).(*model)
	m.SetSize(80, 24)

	sess := &session.Session{
		ID: "test-session",
		Messages: []session.Item{
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:             chat.MessageRoleAssistant,
					ReasoningContent: "Thinking deeply...",
					Content:          "", // No visible content, only reasoning
				},
			}),
		},
	}

	m.LoadFromSession(sess)

	require.Len(t, m.messages, 1)
	assert.Equal(t, types.MessageTypeAssistantReasoning, m.messages[0].Type)
	assert.Equal(t, "Thinking deeply...", m.messages[0].Content)
}
