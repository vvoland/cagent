package session

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
)

func TestTrimMessagesWithToolCalls(t *testing.T) {
	messages := []chat.Message{
		{
			Role:    chat.MessageRoleUser,
			Content: "first message",
		},
		{
			Role:    chat.MessageRoleAssistant,
			Content: "response with tool",
			ToolCalls: []tools.ToolCall{
				{
					ID: "tool1",
				},
			},
		},
		{
			Role:       chat.MessageRoleTool,
			Content:    "tool result",
			ToolCallID: "tool1",
		},
		{
			Role:    chat.MessageRoleUser,
			Content: "second message",
		},
		{
			Role:    chat.MessageRoleAssistant,
			Content: "another response",
			ToolCalls: []tools.ToolCall{
				{
					ID: "tool2",
				},
			},
		},
		{
			Role:       chat.MessageRoleTool,
			Content:    "tool result 2",
			ToolCallID: "tool2",
		},
	}

	// Use 3 as the limit to force trimming
	maxItems := 3

	result := trimMessages(messages, maxItems)

	// Should keep last 3 messages, but ensure tool call consistency
	assert.Len(t, result, maxItems)

	toolCalls := make(map[string]bool)
	for _, msg := range result {
		if msg.Role == chat.MessageRoleAssistant {
			for _, tool := range msg.ToolCalls {
				toolCalls[tool.ID] = true
			}
		}
		if msg.Role == chat.MessageRoleTool {
			assert.True(t, toolCalls[msg.ToolCallID], "tool result should have corresponding assistant message")
		}
	}
}

func TestGetMessagesWithToolCalls(t *testing.T) {
	testAgent := &agent.Agent{}

	s := New()

	s.AddMessage(NewAgentMessage(testAgent, &chat.Message{
		Role:    chat.MessageRoleUser,
		Content: "test message",
	}))

	s.AddMessage(NewAgentMessage(testAgent, &chat.Message{
		Role:    chat.MessageRoleAssistant,
		Content: "using tool",
		ToolCalls: []tools.ToolCall{
			{
				ID: "test-tool",
			},
		},
	}))

	s.AddMessage(NewAgentMessage(testAgent, &chat.Message{
		Role:       chat.MessageRoleTool,
		ToolCallID: "test-tool",
		Content:    "tool result",
	}))

	messages := s.GetMessages(testAgent)

	toolCalls := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role == chat.MessageRoleAssistant {
			for _, tool := range msg.ToolCalls {
				toolCalls[tool.ID] = true
			}
		}
		if msg.Role == chat.MessageRoleTool {
			assert.True(t, toolCalls[msg.ToolCallID], "tool result should have corresponding assistant message")
		}
	}
}

func TestGetMessagesWithSummary(t *testing.T) {
	testAgent := &agent.Agent{}

	s := New()

	s.AddMessage(NewAgentMessage(testAgent, &chat.Message{
		Role:    chat.MessageRoleUser,
		Content: "first message",
	}))
	s.AddMessage(NewAgentMessage(testAgent, &chat.Message{
		Role:    chat.MessageRoleAssistant,
		Content: "first response",
	}))

	s.Messages = append(s.Messages, Item{Summary: "This is a summary of the conversation so far"})

	s.AddMessage(NewAgentMessage(testAgent, &chat.Message{
		Role:    chat.MessageRoleUser,
		Content: "message after summary",
	}))
	s.AddMessage(NewAgentMessage(testAgent, &chat.Message{
		Role:    chat.MessageRoleAssistant,
		Content: "response after summary",
	}))

	messages := s.GetMessages(testAgent)

	// Count non-system messages (user and assistant only)
	userAssistantMessages := 0
	summaryFound := false
	for _, msg := range messages {
		if msg.Role == chat.MessageRoleUser || msg.Role == chat.MessageRoleAssistant {
			userAssistantMessages++
		}
		if msg.Role == chat.MessageRoleSystem && msg.Content == "Session Summary: This is a summary of the conversation so far" {
			summaryFound = true
		}
	}

	// We should have:
	// - 1 summary system message
	// - 2 messages after the summary (user + assistant)
	// - Various other system messages from agent setup
	assert.True(t, summaryFound, "should include summary as system message")
	assert.Equal(t, 2, userAssistantMessages, "should only include messages after summary")
}

func TestGetMessages_Instructions(t *testing.T) {
	testAgent := agent.New("root", "instructions")

	s := New()
	messages := s.GetMessages(testAgent)

	assert.Len(t, messages, 1)
	assert.Equal(t, "instructions", messages[0].Content)
	assert.True(t, messages[0].CacheControl)
}

func TestGetMessages_CacheControl(t *testing.T) {
	testAgent := agent.New("root", "instructions", agent.WithToolSets(&builtin.TodoTool{}))

	s := New()
	messages := s.GetMessages(testAgent)

	assert.Len(t, messages, 2)
	assert.Equal(t, "instructions", messages[0].Content)
	assert.False(t, messages[0].CacheControl)

	assert.Contains(t, messages[1].Content, "Using the Todo Tools")
	assert.True(t, messages[1].CacheControl)
}
