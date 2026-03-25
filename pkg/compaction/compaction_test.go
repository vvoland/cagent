package compaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker-agent/pkg/chat"
	"github.com/docker/docker-agent/pkg/tools"
)

func TestEstimateMessageTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      chat.Message
		expected int64
	}{
		{
			name:     "empty message returns overhead only",
			msg:      chat.Message{},
			expected: 5, // perMessageOverhead
		},
		{
			name:     "text-only message",
			msg:      chat.Message{Content: "Hello, world!"}, // 13 chars → 13/4 = 3 + 5 = 8
			expected: 8,
		},
		{
			name: "multi-content text parts",
			msg: chat.Message{
				MultiContent: []chat.MessagePart{
					{Type: chat.MessagePartTypeText, Text: "first part"},  // 10 chars
					{Type: chat.MessagePartTypeText, Text: "second part"}, // 11 chars
				},
			},
			// 21 total chars → 21/4 = 5 + 5 overhead = 10
			expected: 10,
		},
		{
			name: "message with tool calls",
			msg: chat.Message{
				ToolCalls: []tools.ToolCall{
					{
						Function: tools.FunctionCall{
							Name:      "read_file",                // 9 chars
							Arguments: `{"path":"/tmp/test.txt"}`, // 24 chars
						},
					},
				},
			},
			// 33 chars → 33/4 = 8 + 5 overhead = 13
			expected: 13,
		},
		{
			name: "message with reasoning content",
			msg: chat.Message{
				Content:          "answer",                                         // 6 chars
				ReasoningContent: "Let me think about this carefully step by step", // 47 chars
			},
			// 53 chars → 53/4 = 13 + 5 overhead = 18
			expected: 18,
		},
		{
			name: "combined content types",
			msg: chat.Message{
				Content:          "result",                                   // 6 chars
				ReasoningContent: "thinking",                                 // 8 chars
				MultiContent:     []chat.MessagePart{{Text: "extra detail"}}, // 12 chars
				ToolCalls: []tools.ToolCall{
					{Function: tools.FunctionCall{Name: "cmd", Arguments: `{"x":"y"}`}}, // 3 + 9 = 12 chars
				},
			},
			// 38 total chars → 38/4 = 9 + 5 overhead = 14
			expected: 14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EstimateMessageTokens(&tt.msg)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestShouldCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        int64
		output       int64
		added        int64
		contextLimit int64
		want         bool
	}{
		{
			name:         "below threshold",
			input:        5000,
			output:       2000,
			added:        0,
			contextLimit: 100000,
			want:         false,
		},
		{
			name:         "exactly at 90% boundary",
			input:        90000,
			output:       0,
			added:        0,
			contextLimit: 100000,
			want:         false, // 90000 == int64(100000*0.9), need > not >=
		},
		{
			name:         "just above 90% threshold",
			input:        90001,
			output:       0,
			added:        0,
			contextLimit: 100000,
			want:         true,
		},
		{
			name:         "tool results push past threshold",
			input:        70000,
			output:       10000,
			added:        15000,
			contextLimit: 100000,
			want:         true, // 95000 > 90000
		},
		{
			name:         "zero context limit means unlimited",
			input:        999999,
			output:       999999,
			added:        999999,
			contextLimit: 0,
			want:         false,
		},
		{
			name:         "negative context limit means unlimited",
			input:        999999,
			output:       999999,
			added:        999999,
			contextLimit: -1,
			want:         false,
		},
		{
			name:         "all zeros",
			input:        0,
			output:       0,
			added:        0,
			contextLimit: 100000,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ShouldCompact(tt.input, tt.output, tt.added, tt.contextLimit)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasConversationMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []chat.Message
		want     bool
	}{
		{
			name:     "empty",
			messages: nil,
			want:     false,
		},
		{
			name: "system only",
			messages: []chat.Message{
				{Role: chat.MessageRoleSystem, Content: "You are helpful."},
			},
			want: false,
		},
		{
			name: "system and user",
			messages: []chat.Message{
				{Role: chat.MessageRoleSystem, Content: "You are helpful."},
				{Role: chat.MessageRoleUser, Content: "Hello"},
			},
			want: true,
		},
		{
			name: "only user",
			messages: []chat.Message{
				{Role: chat.MessageRoleUser, Content: "Hello"},
			},
			want: true,
		},
		{
			name: "assistant message",
			messages: []chat.Message{
				{Role: chat.MessageRoleAssistant, Content: "Hi there"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := HasConversationMessages(tt.messages)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	t.Parallel()

	messages := []chat.Message{
		{Role: chat.MessageRoleSystem, Content: "You are helpful."},
		{Role: chat.MessageRoleUser, Content: "Hello", Cost: 0.05},
		{Role: chat.MessageRoleAssistant, Content: "Hi!", Cost: 0.10},
	}

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		out := BuildPrompt(messages, "")

		// Original messages + appended summarization prompt.
		require.Len(t, out, 4)

		// Costs are zeroed.
		for _, msg := range out[:3] {
			assert.Zero(t, msg.Cost, "cost should be zeroed for %q", msg.Content)
		}

		// Last message is the summarization prompt.
		last := out[len(out)-1]
		assert.Equal(t, chat.MessageRoleUser, last.Role)
		assert.Contains(t, last.Content, "summary")
		assert.NotEmpty(t, last.CreatedAt)
	})

	t.Run("with additional prompt", func(t *testing.T) {
		t.Parallel()

		out := BuildPrompt(messages, "focus on code changes")

		last := out[len(out)-1]
		assert.Contains(t, last.Content, "Additional instructions from user: focus on code changes")
	})

	t.Run("does not modify original messages", func(t *testing.T) {
		t.Parallel()

		original := []chat.Message{
			{Role: chat.MessageRoleUser, Content: "Hello", Cost: 0.05},
		}

		_ = BuildPrompt(original, "")

		assert.InDelta(t, 0.05, original[0].Cost, 1e-9)
		assert.Len(t, original, 1)
	})

	t.Run("strips CacheControl from cloned messages", func(t *testing.T) {
		t.Parallel()

		input := []chat.Message{
			{Role: chat.MessageRoleSystem, Content: "system", CacheControl: true},
			{Role: chat.MessageRoleSystem, Content: "context", CacheControl: true},
			{Role: chat.MessageRoleUser, Content: "hello"},
		}

		out := BuildPrompt(input, "")

		// All cloned messages should have CacheControl=false
		for i, msg := range out {
			assert.False(t, msg.CacheControl, "message %d should have CacheControl stripped", i)
		}
		// Original should be unchanged
		assert.True(t, input[0].CacheControl)
		assert.True(t, input[1].CacheControl)
	})
}

func TestPromptsAreEmbedded(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, SystemPrompt, "compaction system prompt should be embedded")
	assert.NotEmpty(t, userPrompt, "compaction user prompt should be embedded")
	assert.Contains(t, SystemPrompt, "summary")
	assert.Contains(t, userPrompt, "summary")
}
