package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker-agent/pkg/chat"
	"github.com/docker/docker-agent/pkg/tools"
)

func TestConvertMessagesToResponseInput_OrphanedFunctionCall(t *testing.T) {
	// Simulate a conversation where an assistant made 2 tool calls but only
	// one has a result (the other was cancelled/interrupted).
	messages := []chat.Message{
		{Role: chat.MessageRoleUser, Content: "hello"},
		{
			Role: chat.MessageRoleAssistant,
			ToolCalls: []tools.ToolCall{
				{ID: "call_1", Type: "function", Function: tools.FunctionCall{Name: "tool_a", Arguments: "{}"}},
				{ID: "call_2", Type: "function", Function: tools.FunctionCall{Name: "tool_b", Arguments: "{}"}},
			},
		},
		{Role: chat.MessageRoleTool, Content: "result a", ToolCallID: "call_1"},
		// call_2 has no result — orphaned
	}

	input := convertMessagesToResponseInput(messages)

	// Count function calls and outputs
	var callIDs, outputIDs []string
	for _, item := range input {
		if item.OfFunctionCall != nil {
			callIDs = append(callIDs, item.OfFunctionCall.CallID)
		}
		if item.OfFunctionCallOutput != nil {
			outputIDs = append(outputIDs, item.OfFunctionCallOutput.CallID)
		}
	}

	require.Len(t, callIDs, 2)
	require.Len(t, outputIDs, 2, "orphaned function call should get a placeholder output")

	assert.Contains(t, outputIDs, "call_1")
	assert.Contains(t, outputIDs, "call_2")
}

func TestConvertMessagesToResponseInput_NoOrphans(t *testing.T) {
	// All tool calls have matching results — no placeholder needed.
	messages := []chat.Message{
		{Role: chat.MessageRoleUser, Content: "hello"},
		{
			Role: chat.MessageRoleAssistant,
			ToolCalls: []tools.ToolCall{
				{ID: "call_1", Type: "function", Function: tools.FunctionCall{Name: "tool_a", Arguments: "{}"}},
			},
		},
		{Role: chat.MessageRoleTool, Content: "result a", ToolCallID: "call_1"},
	}

	input := convertMessagesToResponseInput(messages)

	var outputCount int
	for _, item := range input {
		if item.OfFunctionCallOutput != nil {
			outputCount++
		}
	}
	assert.Equal(t, 1, outputCount, "should not inject extra outputs when all calls have results")
}
