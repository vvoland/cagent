package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/tools"
)

func TestRouge1(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		actual   string
		want     float64
	}{
		{
			name:     "identical strings",
			expected: "the cat sat on the mat",
			actual:   "the cat sat on the mat",
			want:     1.0,
		},
		{
			name:     "completely different strings",
			expected: "hello world",
			actual:   "foo bar baz",
			want:     0.0,
		},
		{
			name:     "partial overlap",
			expected: "the cat sat on the mat",
			actual:   "the cat was on a mat",
			want:     0.6666666666666666, // 4 overlapping words: "the", "cat", "on", "mat"
		},
		{
			name:     "case insensitive",
			expected: "The Cat SAT",
			actual:   "the cat sat",
			want:     1.0,
		},
		{
			name:     "empty expected",
			expected: "",
			actual:   "hello world",
			want:     0.0,
		},
		{
			name:     "empty actual",
			expected: "hello world",
			actual:   "",
			want:     0.0,
		},
		{
			name:     "both empty",
			expected: "",
			actual:   "",
			want:     1.0,
		},
		{
			name:     "repeated words in expected",
			expected: "the the the",
			actual:   "the",
			want:     0.5,
		},
		{
			name:     "repeated words in actual",
			expected: "the",
			actual:   "the the the",
			want:     0.5,
		},
		{
			name:     "single word match",
			expected: "hello",
			actual:   "hello",
			want:     1.0,
		},
		{
			name:     "extra whitespace handled",
			expected: "hello   world",
			actual:   "hello world",
			want:     1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := rouge1(tt.expected, tt.actual)

			assert.InDelta(t, tt.want, got, 0.0001)
		})
	}
}

func TestToolTrajectoryScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected []session.Message
		actual   []session.Message
		want     float64
	}{
		{
			name:     "empty tool messages",
			expected: []session.Message{},
			actual:   []session.Message{},
			want:     1.0,
		},
		{
			name:     "perfect match single tool call",
			expected: []session.Message{msgWithToolCalls("search")},
			actual:   []session.Message{msgWithToolCalls("search")},
			want:     1.0,
		},
		{
			name:     "different tool names",
			expected: []session.Message{msgWithToolCalls("search")},
			actual:   []session.Message{msgWithToolCalls("read_file")},
			want:     0.0,
		},
		{
			name:     "multiple tool calls all match",
			expected: []session.Message{msgWithToolCalls("search", "read_file")},
			actual:   []session.Message{msgWithToolCalls("search", "read_file")},
			want:     1.0,
		},
		{
			name:     "multiple tool calls 1 out of 2 match",
			expected: []session.Message{msgWithToolCalls("search", "read_file")},
			actual:   []session.Message{msgWithToolCalls("search", "write_file")},
			want:     0.5,
		},
		{
			name:     "more expected than actual",
			expected: []session.Message{msgWithToolCalls("search"), msgWithToolCalls("read_file")},
			actual:   []session.Message{msgWithToolCalls("search")},
			want:     0.5,
		},
		{
			name:     "more actual than expected",
			expected: []session.Message{msgWithToolCalls("search")},
			actual:   []session.Message{msgWithToolCalls("search"), msgWithToolCalls("read_file")},
			want:     0.5,
		},
		{
			name:     "multiple messages with multiple tool calls",
			expected: []session.Message{msgWithToolCalls("search", "read_file"), msgWithToolCalls("write_file")},
			actual:   []session.Message{msgWithToolCalls("search", "read_file"), msgWithToolCalls("write_file")},
			want:     1.0,
		},
		{
			name:     "tool call order matters",
			expected: []session.Message{msgWithToolCalls("search", "read_file")},
			actual:   []session.Message{msgWithToolCalls("read_file", "search")},
			want:     0.0,
		},
		{
			name:     "actual has fewer tool calls in message",
			expected: []session.Message{msgWithToolCalls("search", "read_file", "write_file")},
			actual:   []session.Message{msgWithToolCalls("search")},
			want:     1.0 / 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := toolTrajectoryScore(tt.expected, tt.actual)

			assert.InDelta(t, tt.want, got, 0.0001)
		})
	}
}

func msgWithToolCalls(toolNames ...string) session.Message {
	var toolCalls []tools.ToolCall
	for _, name := range toolNames {
		toolCalls = append(toolCalls, tools.ToolCall{
			ID:   "call_" + name,
			Type: "function",
			Function: tools.FunctionCall{
				Name:      name,
				Arguments: "{}",
			},
		})
	}

	return session.Message{
		AgentName: "test_agent",
		Message: chat.Message{
			Role:      chat.MessageRoleAssistant,
			Content:   "",
			ToolCalls: toolCalls,
		},
	}
}
