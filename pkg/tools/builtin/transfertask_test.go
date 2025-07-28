package builtin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskTool(t *testing.T) {
	tool := NewTransferTaskTool()
	assert.NotNil(t, tool)
}

func TestTaskTool_Instructions(t *testing.T) {
	tool := NewTransferTaskTool()
	instructions := tool.Instructions()
	assert.Equal(t, "", instructions, "TaskTool instructions should be empty")
}

func TestTaskTool_Tools(t *testing.T) {
	tool := NewTransferTaskTool()

	tools, err := tool.Tools(context.Background())

	require.NoError(t, err)
	assert.Len(t, tools, 1)

	// Verify transfer_task function
	assert.Equal(t, "transfer_task", tools[0].Function.Name)
	assert.Contains(t, tools[0].Function.Description, "transfer a task to the selected team member")

	// Check parameters
	props := tools[0].Function.Parameters.Properties
	assert.Contains(t, props, "agent")
	assert.Contains(t, props, "task")

	// Check required fields
	assert.Contains(t, tools[0].Function.Parameters.Required, "agent")
	assert.Contains(t, tools[0].Function.Parameters.Required, "task")

	// Verify no handler is provided (it's handled externally)
	assert.Nil(t, tools[0].Handler)
}

func TestTaskTool_StartStop(t *testing.T) {
	tool := NewTransferTaskTool()

	// Test Start method
	err := tool.Start(context.Background())
	require.NoError(t, err)

	// Test Stop method
	err = tool.Stop()
	require.NoError(t, err)
}
