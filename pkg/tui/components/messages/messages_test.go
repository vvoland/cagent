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
	"github.com/docker/cagent/pkg/tui/components/reasoningblock"
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

	// Expect: user message + reasoning block + assistant content = 3 messages
	require.Len(t, m.messages, 3)
	// User message first
	assert.Equal(t, types.MessageTypeUser, m.messages[0].Type)
	assert.Equal(t, "Hello", m.messages[0].Content)
	// Reasoning block second (contains reasoning content)
	assert.Equal(t, types.MessageTypeAssistantReasoningBlock, m.messages[1].Type)
	assert.Equal(t, "Let me think about this...", m.messages[1].Content)
	assert.Equal(t, "root", m.messages[1].Sender)
	// Verify the view is a reasoning block
	block, ok := m.views[1].(*reasoningblock.Model)
	require.True(t, ok, "view should be a reasoning block")
	assert.Equal(t, "Let me think about this...", block.Reasoning())
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

	// Expect: reasoning block (containing reasoning + tool call) + assistant content = 2 messages
	require.Len(t, m.messages, 2)
	// Reasoning block first (contains both reasoning and tool call)
	assert.Equal(t, types.MessageTypeAssistantReasoningBlock, m.messages[0].Type)
	assert.Equal(t, "I should call a tool...", m.messages[0].Content)
	// Verify the view is a reasoning block with tool call
	block, ok := m.views[0].(*reasoningblock.Model)
	require.True(t, ok, "view should be a reasoning block")
	assert.Equal(t, "I should call a tool...", block.Reasoning())
	assert.Equal(t, 1, block.ToolCount(), "block should contain 1 tool call")
	// Assistant content comes last
	assert.Equal(t, types.MessageTypeAssistant, m.messages[1].Type)
	assert.Equal(t, "Tool result processed.", m.messages[1].Content)
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

	// Expect: just the reasoning block (no assistant content)
	require.Len(t, m.messages, 1)
	assert.Equal(t, types.MessageTypeAssistantReasoningBlock, m.messages[0].Type)
	assert.Equal(t, "Thinking deeply...", m.messages[0].Content)
	// Verify the view is a reasoning block
	block, ok := m.views[0].(*reasoningblock.Model)
	require.True(t, ok, "view should be a reasoning block")
	assert.Equal(t, "Thinking deeply...", block.Reasoning())
}

func TestLoadFromSessionToolCallsOnlyNoReasoning(t *testing.T) {
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
					ReasoningContent: "", // No reasoning
					ToolCalls: []tools.ToolCall{
						{ID: "call-1", Function: tools.FunctionCall{Name: "test_tool", Arguments: "{}"}},
					},
					ToolDefinitions: []tools.Tool{
						{Name: "test_tool", Description: "A test tool"},
					},
					Content: "Done.",
				},
			}),
		},
	}

	m.LoadFromSession(sess)

	// Expect: reasoning block (with just tool call) + assistant content = 2 messages
	require.Len(t, m.messages, 2)
	// Reasoning block first (contains tool call, no reasoning)
	assert.Equal(t, types.MessageTypeAssistantReasoningBlock, m.messages[0].Type)
	block, ok := m.views[0].(*reasoningblock.Model)
	require.True(t, ok, "view should be a reasoning block")
	assert.Empty(t, block.Reasoning())
	assert.Equal(t, 1, block.ToolCount(), "block should contain 1 tool call")
	// Assistant content
	assert.Equal(t, types.MessageTypeAssistant, m.messages[1].Type)
	assert.Equal(t, "Done.", m.messages[1].Content)
}

func TestLoadFromSessionWithToolResults(t *testing.T) {
	t.Parallel()

	sessionState := &service.SessionState{}
	m := NewScrollableView(80, 24, sessionState).(*model)
	m.SetSize(80, 24)

	sess := &session.Session{
		ID: "test-session",
		Messages: []session.Item{
			// Assistant message with tool calls
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:             chat.MessageRoleAssistant,
					ReasoningContent: "Let me check this...",
					ToolCalls: []tools.ToolCall{
						{ID: "call-1", Function: tools.FunctionCall{Name: "read_file", Arguments: `{"path": "test.txt"}`}},
						{ID: "call-2", Function: tools.FunctionCall{Name: "list_dir", Arguments: `{"path": "."}`}},
					},
					ToolDefinitions: []tools.Tool{
						{Name: "read_file", Description: "Read a file"},
						{Name: "list_dir", Description: "List directory"},
					},
				},
			}),
			// Tool result for call-1
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:       chat.MessageRoleTool,
					ToolCallID: "call-1",
					Content:    "File content here",
				},
			}),
			// Tool result for call-2
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:       chat.MessageRoleTool,
					ToolCallID: "call-2",
					Content:    "file1.txt\nfile2.txt",
				},
			}),
			// Final assistant response
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:    chat.MessageRoleAssistant,
					Content: "I found the files.",
				},
			}),
		},
	}

	m.LoadFromSession(sess)

	// Expect: reasoning block (reasoning + 2 tool calls with results) + assistant content = 2 messages
	require.Len(t, m.messages, 2)

	// First message should be reasoning block
	assert.Equal(t, types.MessageTypeAssistantReasoningBlock, m.messages[0].Type)
	block, ok := m.views[0].(*reasoningblock.Model)
	require.True(t, ok, "view should be a reasoning block")
	assert.Equal(t, "Let me check this...", block.Reasoning())
	assert.Equal(t, 2, block.ToolCount(), "block should contain 2 tool calls")

	// Expand the block to see tool calls (completed tools are hidden when collapsed)
	block.SetExpanded(true)
	view := block.View()
	assert.Contains(t, view, "read_file", "expanded view should show read_file tool")
	assert.Contains(t, view, "list_dir", "expanded view should show list_dir tool")

	// Second message should be assistant content
	assert.Equal(t, types.MessageTypeAssistant, m.messages[1].Type)
	assert.Equal(t, "I found the files.", m.messages[1].Content)
}

func TestLoadFromSessionCombinesConsecutiveReasoningBlocks(t *testing.T) {
	t.Parallel()

	sessionState := &service.SessionState{}
	m := NewScrollableView(80, 24, sessionState).(*model)
	m.SetSize(80, 24)

	sess := &session.Session{
		ID: "test-session",
		Messages: []session.Item{
			// First assistant message with reasoning and tool call
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:             chat.MessageRoleAssistant,
					ReasoningContent: "First reasoning chunk.",
					ToolCalls: []tools.ToolCall{
						{ID: "call-1", Function: tools.FunctionCall{Name: "tool1", Arguments: "{}"}},
					},
					ToolDefinitions: []tools.Tool{
						{Name: "tool1", Description: "First tool"},
					},
				},
			}),
			// Tool result for call-1
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:       chat.MessageRoleTool,
					ToolCallID: "call-1",
					Content:    "Result 1",
				},
			}),
			// Second assistant message with more reasoning and another tool call (consecutive, no content between)
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:             chat.MessageRoleAssistant,
					ReasoningContent: "Second reasoning chunk.",
					ToolCalls: []tools.ToolCall{
						{ID: "call-2", Function: tools.FunctionCall{Name: "tool2", Arguments: "{}"}},
					},
					ToolDefinitions: []tools.Tool{
						{Name: "tool2", Description: "Second tool"},
					},
				},
			}),
			// Tool result for call-2
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:       chat.MessageRoleTool,
					ToolCallID: "call-2",
					Content:    "Result 2",
				},
			}),
			// Third consecutive reasoning block
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:             chat.MessageRoleAssistant,
					ReasoningContent: "Third reasoning chunk.",
					ToolCalls: []tools.ToolCall{
						{ID: "call-3", Function: tools.FunctionCall{Name: "tool3", Arguments: "{}"}},
					},
					ToolDefinitions: []tools.Tool{
						{Name: "tool3", Description: "Third tool"},
					},
				},
			}),
			// Tool result for call-3
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:       chat.MessageRoleTool,
					ToolCallID: "call-3",
					Content:    "Result 3",
				},
			}),
			// Final assistant response (this breaks the chain)
			session.NewMessageItem(&session.Message{
				AgentName: "root",
				Message: chat.Message{
					Role:    chat.MessageRoleAssistant,
					Content: "All done!",
				},
			}),
		},
	}

	m.LoadFromSession(sess)

	// Should have: 1 combined reasoning block + 1 assistant content = 2 messages
	require.Len(t, m.messages, 2, "consecutive reasoning blocks should be combined into one")

	// First message should be the combined reasoning block
	assert.Equal(t, types.MessageTypeAssistantReasoningBlock, m.messages[0].Type)
	block, ok := m.views[0].(*reasoningblock.Model)
	require.True(t, ok, "view should be a reasoning block")

	// Block should contain all 3 tool calls
	assert.Equal(t, 3, block.ToolCount(), "combined block should contain all 3 tool calls")

	// Reasoning should contain all three chunks
	reasoning := block.Reasoning()
	assert.Contains(t, reasoning, "First reasoning chunk", "should contain first reasoning")
	assert.Contains(t, reasoning, "Second reasoning chunk", "should contain second reasoning")
	assert.Contains(t, reasoning, "Third reasoning chunk", "should contain third reasoning")

	// Expand to verify tools are present
	block.SetExpanded(true)
	view := block.View()
	assert.Contains(t, view, "tool1", "should contain tool1")
	assert.Contains(t, view, "tool2", "should contain tool2")
	assert.Contains(t, view, "tool3", "should contain tool3")

	// Second message should be assistant content
	assert.Equal(t, types.MessageTypeAssistant, m.messages[1].Type)
	assert.Equal(t, "All done!", m.messages[1].Content)
}
