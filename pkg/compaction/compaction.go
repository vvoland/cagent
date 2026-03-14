// Package compaction provides conversation compaction (summarization) for
// chat sessions that approach their model's context window limit.
//
// It is designed as a standalone component that can be used independently of
// the runtime loop. The package exposes:
//
//   - [BuildPrompt]: prepares a conversation for summarization by appending
//     the compaction prompt and sanitizing message costs.
//   - [ShouldCompact]: decides whether a session needs compaction based on
//     token usage and context window limits.
//   - [EstimateMessageTokens]: a fast heuristic for estimating the token
//     count of a single chat message.
//   - [HasConversationMessages]: checks whether a message list contains any
//     non-system messages worth summarizing.
package compaction

import (
	_ "embed"
	"time"

	"github.com/docker/docker-agent/pkg/chat"
)

var (
	//go:embed prompts/compaction-system.txt
	SystemPrompt string

	//go:embed prompts/compaction-user.txt
	userPrompt string
)

// contextThreshold is the fraction of the context window at which compaction
// is triggered. When the estimated token usage exceeds this fraction of the
// context limit, compaction is recommended.
const contextThreshold = 0.9

// Result holds the outcome of a compaction operation.
type Result struct {
	// Summary is the generated summary text.
	Summary string

	// InputTokens is the token count reported by the summarization model,
	// used as an approximation of the new context size after compaction.
	InputTokens int64

	// Cost is the cost of the summarization request in dollars.
	Cost float64
}

// BuildPrompt prepares the messages for a summarization request.
// It clones the conversation (zeroing per-message costs so they aren't
// double-counted), then appends a user message containing the compaction
// prompt. If additionalPrompt is non-empty it is included as extra
// instructions.
//
// Callers should first check [HasConversationMessages] to avoid sending
// an empty conversation to the model.
func BuildPrompt(messages []chat.Message, additionalPrompt string) []chat.Message {
	prompt := userPrompt
	if additionalPrompt != "" {
		prompt += "\n\nAdditional instructions from user: " + additionalPrompt
	}

	out := make([]chat.Message, len(messages), len(messages)+1)
	for i, msg := range messages {
		cloned := msg
		cloned.Cost = 0
		out[i] = cloned
	}
	out = append(out, chat.Message{
		Role:      chat.MessageRoleUser,
		Content:   prompt,
		CreatedAt: time.Now().Format(time.RFC3339),
	})

	return out
}

// ShouldCompact reports whether a session's context usage has crossed the
// compaction threshold. It returns true when the estimated total token count
// (input + output + addedTokens) exceeds [contextThreshold] (90%) of
// contextLimit. A non-positive contextLimit is treated as unlimited and
// always returns false.
func ShouldCompact(inputTokens, outputTokens, addedTokens, contextLimit int64) bool {
	if contextLimit <= 0 {
		return false
	}
	estimated := inputTokens + outputTokens + addedTokens
	return estimated > int64(float64(contextLimit)*contextThreshold)
}

// EstimateMessageTokens returns a rough token-count estimate for a single
// chat message based on its text length. This is intentionally conservative
// (overestimates) so that proactive compaction fires before we hit the limit.
//
// The estimate accounts for message content, multi-content text parts,
// reasoning content, tool call arguments, and a small per-message overhead
// for role/metadata tokens.
func EstimateMessageTokens(msg *chat.Message) int64 {
	// charsPerToken: average characters per token. 4 is a widely-used
	// heuristic for English; slightly overestimates for code/JSON (~3.5).
	const charsPerToken = 4

	// perMessageOverhead: role, ToolCallID, delimiters, etc.
	const perMessageOverhead = 5

	var chars int
	chars += len(msg.Content)
	for _, part := range msg.MultiContent {
		chars += len(part.Text)
	}
	chars += len(msg.ReasoningContent)
	for _, tc := range msg.ToolCalls {
		chars += len(tc.Function.Arguments)
		chars += len(tc.Function.Name)
	}

	if chars == 0 {
		return perMessageOverhead
	}
	return int64(chars/charsPerToken) + perMessageOverhead
}

// HasConversationMessages reports whether messages contains at least one
// non-system message. A session with only system prompts has no conversation
// to summarize.
func HasConversationMessages(messages []chat.Message) bool {
	for _, msg := range messages {
		if msg.Role != chat.MessageRoleSystem {
			return true
		}
	}
	return false
}
