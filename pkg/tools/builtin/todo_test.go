package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/tools"
)

func TestTodoTool_DisplayNames(t *testing.T) {
	tool := NewTodoTool()

	all, err := tool.Tools(t.Context())
	require.NoError(t, err)

	for _, tool := range all {
		assert.NotEmpty(t, tool.DisplayName())
		assert.NotEqual(t, tool.Name, tool.DisplayName())
	}
}

func TestTodoTool_CreateTodo(t *testing.T) {
	tool := NewTodoTool()

	result, err := tool.handler.createTodo(t.Context(), CreateTodoArgs{
		Description: "Test todo item",
	})

	require.NoError(t, err)
	assert.Contains(t, result.Output, "Created todo [todo_1]: Test todo item")

	assert.Equal(t, 1, tool.handler.todos.Length())
	todos := tool.handler.todos.All()
	require.Len(t, todos, 1)
	assert.Equal(t, "todo_1", todos[0].ID)
	assert.Equal(t, "Test todo item", todos[0].Description)
	assert.Equal(t, "pending", todos[0].Status)

	// Verify Meta contains the created todo
	metaTodos, ok := result.Meta.([]Todo)
	require.True(t, ok, "Meta should be []Todo")
	require.Len(t, metaTodos, 1)
	assert.Equal(t, "todo_1", metaTodos[0].ID)
	assert.Equal(t, "Test todo item", metaTodos[0].Description)
	assert.Equal(t, "pending", metaTodos[0].Status)
}

func TestTodoTool_CreateTodos(t *testing.T) {
	tool := NewTodoTool()

	result, err := tool.handler.createTodos(t.Context(), CreateTodosArgs{
		Descriptions: []string{
			"First todo item",
			"Second todo item",
			"Third todo item",
		},
	})

	require.NoError(t, err)
	assert.Contains(t, result.Output, "Created 3 todos:")
	assert.Contains(t, result.Output, "todo_1")
	assert.Contains(t, result.Output, "todo_2")
	assert.Contains(t, result.Output, "todo_3")

	assert.Equal(t, 3, tool.handler.todos.Length())

	// Verify Meta contains all todos (order not guaranteed from map)
	metaTodos, ok := result.Meta.([]Todo)
	require.True(t, ok, "Meta should be []Todo")
	require.Len(t, metaTodos, 3)

	// Verify order is preserved
	assert.Equal(t, "First todo item", metaTodos[0].Description)
	assert.Equal(t, "Second todo item", metaTodos[1].Description)
	assert.Equal(t, "Third todo item", metaTodos[2].Description)

	result, err = tool.handler.createTodos(t.Context(), CreateTodosArgs{
		Descriptions: []string{
			"Last todo item",
		},
	})

	require.NoError(t, err)
	assert.Contains(t, result.Output, "Created 1 todos:")
	assert.Contains(t, result.Output, "todo_4")
	assert.Equal(t, 4, tool.handler.todos.Length())

	// Verify Meta for second call contains all 4 todos
	metaTodos, ok = result.Meta.([]Todo)
	require.True(t, ok, "Meta should be []Todo")
	require.Len(t, metaTodos, 4)
}

func TestTodoTool_UpdateTodo(t *testing.T) {
	tool := NewTodoTool()

	_, err := tool.handler.createTodo(t.Context(), CreateTodoArgs{
		Description: "Test todo item",
	})
	require.NoError(t, err)

	result, err := tool.handler.updateTodo(t.Context(), UpdateTodoArgs{
		ID:     "todo_1",
		Status: "completed",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "Updated todo [todo_1] to status: [completed]")

	todos := tool.handler.todos.All()
	require.Len(t, todos, 1)
	assert.Equal(t, "completed", todos[0].Status)

	// Verify Meta contains all todos with updated status
	metaTodos, ok := result.Meta.([]Todo)
	require.True(t, ok, "Meta should be []Todo")
	require.Len(t, metaTodos, 1)
	assert.Equal(t, "todo_1", metaTodos[0].ID)
	assert.Equal(t, "Test todo item", metaTodos[0].Description)
	assert.Equal(t, "completed", metaTodos[0].Status)
}

func TestTodoTool_ListTodos(t *testing.T) {
	tool := NewTodoTool()

	todos := []string{
		"First todo item",
		"Second todo item",
		"Third todo item",
	}

	for _, todoDesc := range todos {
		_, err := tool.handler.createTodo(t.Context(), CreateTodoArgs{
			Description: todoDesc,
		})

		require.NoError(t, err)
	}

	result, err := tool.handler.listTodos(t.Context(), nil)

	require.NoError(t, err)
	assert.Contains(t, result.Output, "Current todos:")
	for _, todoDesc := range todos {
		assert.Contains(t, result.Output, todoDesc)
		assert.Contains(t, result.Output, "Status: pending")
	}

	// Verify Meta contains all todos
	metaTodos, ok := result.Meta.([]Todo)
	require.True(t, ok, "Meta should be []Todo")
	require.Len(t, metaTodos, 3)
}

func TestTodoTool_UpdateNonexistentTodo(t *testing.T) {
	tool := NewTodoTool()

	res, err := tool.handler.updateTodo(t.Context(), UpdateTodoArgs{
		ID:     "nonexistent_todo",
		Status: "completed",
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
}

func TestTodoTool_OutputSchema(t *testing.T) {
	tool := NewTodoTool()

	allTools, err := tool.Tools(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, allTools)

	for _, tool := range allTools {
		assert.NotNil(t, tool.OutputSchema)
	}
}

func TestTodoTool_ParametersAreObjects(t *testing.T) {
	tool := NewTodoTool()

	allTools, err := tool.Tools(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, allTools)

	for _, tool := range allTools {
		m, err := tools.SchemaToMap(tool.Parameters)

		require.NoError(t, err)
		assert.Equal(t, "object", m["type"])
	}
}
