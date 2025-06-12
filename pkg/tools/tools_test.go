package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolTypes(t *testing.T) {
	// Test basic structure of ToolCall
	toolCall := ToolCall{
		ID: "tool_123",
		Function: FunctionCall{
			Name:      "test_function",
			Arguments: `{"arg1": "value1"}`,
		},
	}

	assert.Equal(t, "tool_123", toolCall.ID)
	assert.Equal(t, "test_function", toolCall.Function.Name)
	assert.Equal(t, `{"arg1": "value1"}`, toolCall.Function.Arguments)

	// Test ToolCallResult
	result := ToolCallResult{
		Output: "test output",
	}

	assert.Equal(t, "test output", result.Output)

	// Test FunctionDefinition
	funcDef := FunctionDefinition{
		Name:        "test_function",
		Description: "A test function",
		Strict:      true,
		Parameters: FunctionParamaters{
			Type: "object",
			Properties: map[string]any{
				"arg1": map[string]any{
					"type":        "string",
					"description": "First argument",
				},
			},
			Required: []string{"arg1"},
		},
	}

	assert.Equal(t, "test_function", funcDef.Name)
	assert.Equal(t, "A test function", funcDef.Description)
	assert.True(t, funcDef.Strict)
	assert.Equal(t, "object", funcDef.Parameters.Type)
	assert.Contains(t, funcDef.Parameters.Properties, "arg1")
	assert.Contains(t, funcDef.Parameters.Required, "arg1")

	// Test Tool
	mockHandler := func(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
		return &ToolCallResult{Output: "test"}, nil
	}

	tool := Tool{
		Function: &funcDef,
		Handler:  mockHandler,
	}

	assert.Equal(t, &funcDef, tool.Function)
	assert.NotNil(t, tool.Handler)

	// Test handler functionality
	testCtx := context.Background()
	testToolCall := ToolCall{
		Function: FunctionCall{
			Name:      "test_function",
			Arguments: "{}",
		},
	}

	var handlerResult ToolCallResult
	tmp, err := tool.Handler(testCtx, testToolCall)
	assert.NoError(t, err)
	handlerResult = *tmp
	assert.Equal(t, "test", handlerResult.Output)
}
