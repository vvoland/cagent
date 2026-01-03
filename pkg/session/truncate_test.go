package session

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
)

func TestTruncateOldToolContent(t *testing.T) {
	t.Run("keeps recent tool content within budget", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.MessageRoleUser, Content: "hello"},
			{
				Role: chat.MessageRoleAssistant,
				ToolCalls: []tools.ToolCall{
					{ID: "1", Function: tools.FunctionCall{Name: "test", Arguments: `{"key":"value"}`}},
				},
			},
			{Role: chat.MessageRoleTool, ToolCallID: "1", Content: "result"},
		}

		result := truncateOldToolContent(messages, 1000)

		assert.JSONEq(t, `{"key":"value"}`, result[1].ToolCalls[0].Function.Arguments)
		assert.Equal(t, "result", result[2].Content)
	})

	t.Run("truncates oldest tool content when budget exceeded", func(t *testing.T) {
		oldArgs := strings.Repeat("a", 400)   // 100 tokens
		oldResult := strings.Repeat("b", 400) // 100 tokens
		newArgs := strings.Repeat("c", 200)   // 50 tokens
		newResult := strings.Repeat("d", 200) // 50 tokens

		messages := []chat.Message{
			{Role: chat.MessageRoleUser, Content: "first"},
			{
				Role: chat.MessageRoleAssistant,
				ToolCalls: []tools.ToolCall{
					{ID: "old", Function: tools.FunctionCall{Name: "test", Arguments: oldArgs}},
				},
			},
			{Role: chat.MessageRoleTool, ToolCallID: "old", Content: oldResult},
			{Role: chat.MessageRoleUser, Content: "second"},
			{
				Role: chat.MessageRoleAssistant,
				ToolCalls: []tools.ToolCall{
					{ID: "new", Function: tools.FunctionCall{Name: "test", Arguments: newArgs}},
				},
			},
			{Role: chat.MessageRoleTool, ToolCallID: "new", Content: newResult},
		}

		// Budget of 120 tokens: new content (100 tokens) fits, old content gets truncated
		result := truncateOldToolContent(messages, 120)

		// New content should be preserved
		assert.Equal(t, newArgs, result[4].ToolCalls[0].Function.Arguments)
		assert.Equal(t, newResult, result[5].Content)

		// Old content should be truncated
		assert.Equal(t, toolContentPlaceholder, result[1].ToolCalls[0].Function.Arguments)
		assert.Equal(t, toolContentPlaceholder, result[2].Content)
	})

	t.Run("does not modify non-tool messages", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.MessageRoleUser, Content: strings.Repeat("x", 1000)},
			{Role: chat.MessageRoleAssistant, Content: strings.Repeat("y", 1000)},
			{Role: chat.MessageRoleSystem, Content: strings.Repeat("z", 1000)},
		}

		result := truncateOldToolContent(messages, 10)

		assert.Equal(t, messages[0].Content, result[0].Content)
		assert.Equal(t, messages[1].Content, result[1].Content)
		assert.Equal(t, messages[2].Content, result[2].Content)
	})

	t.Run("handles multiple tool calls in single message", func(t *testing.T) {
		args1 := strings.Repeat("a", 200) // 50 tokens
		args2 := strings.Repeat("b", 200) // 50 tokens
		args3 := strings.Repeat("c", 200) // 50 tokens

		messages := []chat.Message{
			{
				Role: chat.MessageRoleAssistant,
				ToolCalls: []tools.ToolCall{
					{ID: "1", Function: tools.FunctionCall{Name: "test1", Arguments: args1}},
					{ID: "2", Function: tools.FunctionCall{Name: "test2", Arguments: args2}},
					{ID: "3", Function: tools.FunctionCall{Name: "test3", Arguments: args3}},
				},
			},
			{Role: chat.MessageRoleTool, ToolCallID: "1", Content: "r1"},
			{Role: chat.MessageRoleTool, ToolCallID: "2", Content: "r2"},
			{Role: chat.MessageRoleTool, ToolCallID: "3", Content: "r3"},
		}

		// Budget of 60 tokens: only the last tool call args fit
		result := truncateOldToolContent(messages, 60)

		// Last tool call should be preserved
		assert.Equal(t, args3, result[0].ToolCalls[2].Function.Arguments)
		// Earlier tool calls should be truncated
		assert.Equal(t, toolContentPlaceholder, result[0].ToolCalls[0].Function.Arguments)
		assert.Equal(t, toolContentPlaceholder, result[0].ToolCalls[1].Function.Arguments)
	})

	t.Run("returns original messages when maxTokens is zero", func(t *testing.T) {
		messages := []chat.Message{
			{
				Role: chat.MessageRoleAssistant,
				ToolCalls: []tools.ToolCall{
					{ID: "1", Function: tools.FunctionCall{Name: "test", Arguments: "args"}},
				},
			},
			{Role: chat.MessageRoleTool, ToolCallID: "1", Content: "result"},
		}

		result := truncateOldToolContent(messages, 0)

		assert.Equal(t, messages, result)
	})

	t.Run("returns original messages when maxTokens is negative", func(t *testing.T) {
		messages := []chat.Message{
			{
				Role: chat.MessageRoleAssistant,
				ToolCalls: []tools.ToolCall{
					{ID: "1", Function: tools.FunctionCall{Name: "test", Arguments: "args"}},
				},
			},
			{Role: chat.MessageRoleTool, ToolCallID: "1", Content: "result"},
		}

		result := truncateOldToolContent(messages, -10)

		assert.Equal(t, messages, result)
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		originalArgs := strings.Repeat("x", 400)
		messages := []chat.Message{
			{
				Role: chat.MessageRoleAssistant,
				ToolCalls: []tools.ToolCall{
					{ID: "1", Function: tools.FunctionCall{Name: "test", Arguments: originalArgs}},
				},
			},
			{Role: chat.MessageRoleTool, ToolCallID: "1", Content: strings.Repeat("y", 400)},
		}

		result := truncateOldToolContent(messages, 10)

		// Result should be truncated
		assert.Equal(t, toolContentPlaceholder, result[0].ToolCalls[0].Function.Arguments)
		assert.Equal(t, toolContentPlaceholder, result[1].Content)

		// Original should be unchanged
		assert.Equal(t, originalArgs, messages[0].ToolCalls[0].Function.Arguments)
	})

	t.Run("handles empty messages slice", func(t *testing.T) {
		result := truncateOldToolContent(nil, 1000)
		assert.Nil(t, result)

		result = truncateOldToolContent([]chat.Message{}, 1000)
		require.NotNil(t, result)
		assert.Empty(t, result)
	})
}
