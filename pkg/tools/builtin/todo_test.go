package builtin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker-agent/pkg/tools"
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
	storage := NewMemoryTodoStorage()
	tool := NewTodoTool(WithStorage(storage))

	result, err := tool.handler.createTodo(t.Context(), CreateTodoArgs{
		Description: "Test todo item",
	})
	require.NoError(t, err)

	var output Todo
	require.NoError(t, json.Unmarshal([]byte(result.Output), &output))
	assert.Equal(t, "todo_1", output.ID)
	assert.Equal(t, "Test todo item", output.Description)
	assert.Equal(t, "pending", output.Status)

	require.Equal(t, 1, storage.Len())
	requireMeta(t, result, 1)
}

func TestTodoTool_CreateTodos(t *testing.T) {
	storage := NewMemoryTodoStorage()
	tool := NewTodoTool(WithStorage(storage))

	result, err := tool.handler.createTodos(t.Context(), CreateTodosArgs{
		Descriptions: []string{"First", "Second", "Third"},
	})
	require.NoError(t, err)

	var output CreateTodosOutput
	require.NoError(t, json.Unmarshal([]byte(result.Output), &output))
	require.Len(t, output.Created, 3)
	assert.Equal(t, "todo_1", output.Created[0].ID)
	assert.Equal(t, "First", output.Created[0].Description)
	assert.Equal(t, "pending", output.Created[0].Status)
	assert.Equal(t, "todo_2", output.Created[1].ID)
	assert.Equal(t, "todo_3", output.Created[2].ID)

	assert.Equal(t, 3, storage.Len())
	requireMeta(t, result, 3)

	// A second call continues the ID sequence
	result, err = tool.handler.createTodos(t.Context(), CreateTodosArgs{
		Descriptions: []string{"Last"},
	})
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal([]byte(result.Output), &output))
	require.Len(t, output.Created, 1)
	assert.Equal(t, "todo_4", output.Created[0].ID)
	assert.Equal(t, 4, storage.Len())
	requireMeta(t, result, 4)
}

func TestTodoTool_ListTodos(t *testing.T) {
	tool := NewTodoTool()

	descs := []string{"First", "Second", "Third"}
	for _, d := range descs {
		_, err := tool.handler.createTodo(t.Context(), CreateTodoArgs{Description: d})
		require.NoError(t, err)
	}

	result, err := tool.handler.listTodos(t.Context(), tools.ToolCall{})
	require.NoError(t, err)

	var output ListTodosOutput
	require.NoError(t, json.Unmarshal([]byte(result.Output), &output))
	require.Len(t, output.Todos, 3)
	for i, d := range descs {
		assert.Equal(t, d, output.Todos[i].Description)
		assert.Equal(t, "pending", output.Todos[i].Status)
	}

	requireMeta(t, result, 3)
}

func TestTodoTool_UpdateTodos(t *testing.T) {
	storage := NewMemoryTodoStorage()
	tool := NewTodoTool(WithStorage(storage))

	_, err := tool.handler.createTodos(t.Context(), CreateTodosArgs{
		Descriptions: []string{"First", "Second", "Third"},
	})
	require.NoError(t, err)

	result, err := tool.handler.updateTodos(t.Context(), UpdateTodosArgs{
		Updates: []TodoUpdate{
			{ID: "todo_1", Status: "completed"},
			{ID: "todo_3", Status: "in-progress"},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var output UpdateTodosOutput
	require.NoError(t, json.Unmarshal([]byte(result.Output), &output))
	require.Len(t, output.Updated, 2)
	assert.Equal(t, "todo_1", output.Updated[0].ID)
	assert.Equal(t, "completed", output.Updated[0].Status)
	assert.Equal(t, "todo_3", output.Updated[1].ID)
	assert.Equal(t, "in-progress", output.Updated[1].Status)
	assert.Empty(t, output.NotFound)

	todos := storage.All()
	require.Len(t, todos, 3)
	assert.Equal(t, "completed", todos[0].Status)
	assert.Equal(t, "pending", todos[1].Status)
	assert.Equal(t, "in-progress", todos[2].Status)

	requireMeta(t, result, 3)
}

func TestTodoTool_UpdateTodos_PartialFailure(t *testing.T) {
	storage := NewMemoryTodoStorage()
	tool := NewTodoTool(WithStorage(storage))

	_, err := tool.handler.createTodos(t.Context(), CreateTodosArgs{
		Descriptions: []string{"First", "Second"},
	})
	require.NoError(t, err)

	result, err := tool.handler.updateTodos(t.Context(), UpdateTodosArgs{
		Updates: []TodoUpdate{
			{ID: "todo_1", Status: "completed"},
			{ID: "nonexistent", Status: "completed"},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var output UpdateTodosOutput
	require.NoError(t, json.Unmarshal([]byte(result.Output), &output))
	require.Len(t, output.Updated, 1)
	assert.Equal(t, "todo_1", output.Updated[0].ID)
	require.Len(t, output.NotFound, 1)
	assert.Equal(t, "nonexistent", output.NotFound[0])

	todos := storage.All()
	require.Len(t, todos, 2)
	assert.Equal(t, "completed", todos[0].Status)
	assert.Equal(t, "pending", todos[1].Status)
}

func TestTodoTool_UpdateTodos_AllNotFound(t *testing.T) {
	tool := NewTodoTool()

	result, err := tool.handler.updateTodos(t.Context(), UpdateTodosArgs{
		Updates: []TodoUpdate{
			{ID: "nonexistent1", Status: "completed"},
			{ID: "nonexistent2", Status: "completed"},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)

	var output UpdateTodosOutput
	require.NoError(t, json.Unmarshal([]byte(result.Output), &output))
	assert.Empty(t, output.Updated)
	require.Len(t, output.NotFound, 2)
	assert.Equal(t, "nonexistent1", output.NotFound[0])
	assert.Equal(t, "nonexistent2", output.NotFound[1])
}

func TestTodoTool_UpdateTodos_ClearsWhenAllCompleted(t *testing.T) {
	storage := NewMemoryTodoStorage()
	tool := NewTodoTool(WithStorage(storage))

	_, err := tool.handler.createTodos(t.Context(), CreateTodosArgs{
		Descriptions: []string{"First", "Second"},
	})
	require.NoError(t, err)

	result, err := tool.handler.updateTodos(t.Context(), UpdateTodosArgs{
		Updates: []TodoUpdate{
			{ID: "todo_1", Status: "completed"},
			{ID: "todo_2", Status: "completed"},
		},
	})
	require.NoError(t, err)

	var output UpdateTodosOutput
	require.NoError(t, json.Unmarshal([]byte(result.Output), &output))
	require.Len(t, output.Updated, 2)

	assert.Empty(t, storage.All())
	requireMeta(t, result, 0)
}

func TestTodoTool_WithStorage(t *testing.T) {
	storage := NewMemoryTodoStorage()
	tool := NewTodoTool(WithStorage(storage))

	_, err := tool.handler.createTodo(t.Context(), CreateTodoArgs{Description: "Test item"})
	require.NoError(t, err)

	assert.Equal(t, 1, storage.Len())
	assert.Equal(t, "Test item", storage.All()[0].Description)
}

func TestTodoTool_WithStorage_NilPanics(t *testing.T) {
	assert.Panics(t, func() {
		WithStorage(nil)
	})
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

// requireMeta asserts that result.Meta is a []Todo of the expected length.
func requireMeta(t *testing.T, result *tools.ToolCallResult, expectedLen int) {
	t.Helper()
	metaTodos, ok := result.Meta.([]Todo)
	require.True(t, ok, "Meta should be []Todo")
	require.Len(t, metaTodos, expectedLen)
}
