package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewThinkTool(t *testing.T) {
	tool := NewThinkTool()
	
	assert.NotNil(t, tool)
	assert.NotNil(t, tool.handler)
	assert.Empty(t, tool.handler.thoughts)
}

func TestThinkTool_Instructions(t *testing.T) {
	tool := NewThinkTool()
	
	instructions := tool.Instructions()
	assert.Contains(t, instructions, "Using the think tool")
	assert.Contains(t, instructions, "Use the think tool generously")
}

func TestThinkTool_Tools(t *testing.T) {
	tool := NewThinkTool()
	
	tools, err := tool.Tools(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, tools, 1)
	
	// Verify think function
	assert.Equal(t, "think", tools[0].Function.Name)
	assert.Contains(t, tools[0].Function.Description, "Use the tool to think about something")
	
	// Check parameters
	props := tools[0].Function.Parameters.Properties
	assert.Contains(t, props, "thought")
	
	// Check required fields
	assert.Contains(t, tools[0].Function.Parameters.Required, "thought")
	
	// Verify handler is provided
	assert.NotNil(t, tools[0].Handler)
}

func TestThinkTool_Handler(t *testing.T) {
	tool := NewThinkTool()
	
	// Get handler from tool
	tools, err := tool.Tools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	
	handler := tools[0].Handler
	
	// Create tool call with thought
	args := struct {
		Thought string `json:"thought"`
	}{
		Thought: "This is a test thought",
	}
	argsBytes, _ := json.Marshal(args)
	
	toolCall := ToolCall{
		Function: FunctionCall{
			Name:      "think",
			Arguments: string(argsBytes),
		},
	}
	
	// Call handler
	result, err := handler(context.Background(), toolCall)
	
	// Verify
	require.NoError(t, err)
	assert.Contains(t, result.Output, "This is a test thought")
	
	// Add another thought
	args.Thought = "Another thought"
	argsBytes, _ = json.Marshal(args)
	
	toolCall.Function.Arguments = string(argsBytes)
	
	result, err = handler(context.Background(), toolCall)
	
	// Verify both thoughts are in output
	require.NoError(t, err)
	assert.Contains(t, result.Output, "This is a test thought")
	assert.Contains(t, result.Output, "Another thought")
}

func TestThinkTool_InvalidArguments(t *testing.T) {
	tool := NewThinkTool()
	
	// Get handler from tool
	tools, err := tool.Tools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	
	handler := tools[0].Handler
	
	// Invalid JSON
	toolCall := ToolCall{
		Function: FunctionCall{
			Name:      "think",
			Arguments: "{invalid json",
		},
	}
	
	result, err := handler(context.Background(), toolCall)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestThinkTool_StartStop(t *testing.T) {
	tool := NewThinkTool()
	
	// Test Start method
	err := tool.Start(context.Background())
	assert.NoError(t, err)
	
	// Test Stop method
	err = tool.Stop()
	assert.NoError(t, err)
}