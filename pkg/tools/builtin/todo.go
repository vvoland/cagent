package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/docker/docker-agent/pkg/concurrent"
	"github.com/docker/docker-agent/pkg/tools"
)

const (
	ToolNameCreateTodo  = "create_todo"
	ToolNameCreateTodos = "create_todos"
	ToolNameUpdateTodos = "update_todos"
	ToolNameListTodos   = "list_todos"
)

type TodoTool struct {
	handler *todoHandler
}

// Verify interface compliance
var (
	_ tools.ToolSet      = (*TodoTool)(nil)
	_ tools.Instructable = (*TodoTool)(nil)
)

type Todo struct {
	ID          string `json:"id" jsonschema:"ID of the todo item"`
	Description string `json:"description" jsonschema:"Description of the todo item"`
	Status      string `json:"status" jsonschema:"Status of the todo item (pending, in-progress, completed)"`
}

type CreateTodoArgs struct {
	Description string `json:"description" jsonschema:"Description of the todo item"`
}

type CreateTodosArgs struct {
	Descriptions []string `json:"descriptions" jsonschema:"Descriptions of the todo items"`
}

type TodoUpdate struct {
	ID     string `json:"id" jsonschema:"ID of the todo item"`
	Status string `json:"status" jsonschema:"New status (pending, in-progress, completed)"`
}

type UpdateTodosArgs struct {
	Updates []TodoUpdate `json:"updates" jsonschema:"List of todo updates"`
}

// Output types for JSON-structured responses.

type CreateTodosOutput struct {
	Created []Todo `json:"created" jsonschema:"List of created todo items"`
}

type UpdateTodosOutput struct {
	Updated  []TodoUpdate `json:"updated,omitempty" jsonschema:"List of successfully updated todos"`
	NotFound []string     `json:"not_found,omitempty" jsonschema:"IDs of todos that were not found"`
}

type ListTodosOutput struct {
	Todos []Todo `json:"todos" jsonschema:"List of all current todo items"`
}

// TodoStorage defines the storage layer for todo items.
type TodoStorage interface {
	// Add appends a new todo item.
	Add(todo Todo)
	// All returns a copy of all todo items.
	All() []Todo
	// Len returns the number of todo items.
	Len() int
	// FindByID returns the index of the todo with the given ID, or -1 if not found.
	FindByID(id string) int
	// Update modifies the todo at the given index using the provided function.
	Update(index int, fn func(Todo) Todo)
	// Clear removes all todo items.
	Clear()
}

// MemoryTodoStorage is an in-memory, concurrency-safe implementation of TodoStorage.
type MemoryTodoStorage struct {
	todos *concurrent.Slice[Todo]
}

func NewMemoryTodoStorage() *MemoryTodoStorage {
	return &MemoryTodoStorage{
		todos: concurrent.NewSlice[Todo](),
	}
}

func (s *MemoryTodoStorage) Add(todo Todo) {
	s.todos.Append(todo)
}

func (s *MemoryTodoStorage) All() []Todo {
	return s.todos.All()
}

func (s *MemoryTodoStorage) Len() int {
	return s.todos.Length()
}

func (s *MemoryTodoStorage) FindByID(id string) int {
	_, idx := s.todos.Find(func(t Todo) bool { return t.ID == id })
	return idx
}

func (s *MemoryTodoStorage) Update(index int, fn func(Todo) Todo) {
	s.todos.Update(index, fn)
}

func (s *MemoryTodoStorage) Clear() {
	s.todos.Clear()
}

// TodoOption is a functional option for configuring a TodoTool.
type TodoOption func(*TodoTool)

// WithStorage sets a custom storage implementation for the TodoTool.
// The provided storage must not be nil.
func WithStorage(storage TodoStorage) TodoOption {
	if storage == nil {
		panic("todo: storage must not be nil")
	}
	return func(t *TodoTool) {
		t.handler.storage = storage
	}
}

type todoHandler struct {
	storage TodoStorage
	nextID  atomic.Int64
}

var NewSharedTodoTool = sync.OnceValue(func() *TodoTool { return NewTodoTool() })

func NewTodoTool(opts ...TodoOption) *TodoTool {
	t := &TodoTool{
		handler: &todoHandler{
			storage: NewMemoryTodoStorage(),
		},
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *TodoTool) Instructions() string {
	return `## Using the Todo Tools

IMPORTANT: You MUST use these tools to track the progress of your tasks:

1. Before starting any complex task:
	- Create a todo for each major step using create_todo
	- Break down complex steps into smaller todos

2. While working:
	- Use list_todos frequently to keep track of remaining work
	- Mark todos as "completed" when finished

3. Task Management Rules:
	- Never start a new task without creating a todo for it
	- Always check list_todos before responding to ensure no steps are missed
	- Update todo status to reflect current progress

This toolset is REQUIRED for maintaining task state and ensuring all steps are completed.`
}

// addTodo creates a new todo and adds it to storage.
func (h *todoHandler) addTodo(description string) Todo {
	todo := Todo{
		ID:          fmt.Sprintf("todo_%d", h.nextID.Add(1)),
		Description: description,
		Status:      "pending",
	}
	h.storage.Add(todo)
	return todo
}

// jsonResult builds a ToolCallResult with a JSON-serialized output and current storage as Meta.
func (h *todoHandler) jsonResult(v any) (*tools.ToolCallResult, error) {
	out, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshaling todo output: %w", err)
	}
	return &tools.ToolCallResult{
		Output: string(out),
		Meta:   h.storage.All(),
	}, nil
}

func (h *todoHandler) createTodo(_ context.Context, params CreateTodoArgs) (*tools.ToolCallResult, error) {
	return h.jsonResult(h.addTodo(params.Description))
}

func (h *todoHandler) createTodos(_ context.Context, params CreateTodosArgs) (*tools.ToolCallResult, error) {
	created := make([]Todo, 0, len(params.Descriptions))
	for _, desc := range params.Descriptions {
		created = append(created, h.addTodo(desc))
	}
	return h.jsonResult(CreateTodosOutput{Created: created})
}

func (h *todoHandler) updateTodos(_ context.Context, params UpdateTodosArgs) (*tools.ToolCallResult, error) {
	result := UpdateTodosOutput{}

	for _, update := range params.Updates {
		idx := h.storage.FindByID(update.ID)
		if idx == -1 {
			result.NotFound = append(result.NotFound, update.ID)
			continue
		}

		h.storage.Update(idx, func(t Todo) Todo {
			t.Status = update.Status
			return t
		})
		result.Updated = append(result.Updated, update)
	}

	if len(result.NotFound) > 0 && len(result.Updated) == 0 {
		res, err := h.jsonResult(result)
		if err != nil {
			return nil, err
		}
		res.IsError = true
		return res, nil
	}

	if h.allCompleted() {
		h.storage.Clear()
	}

	return h.jsonResult(result)
}

func (h *todoHandler) allCompleted() bool {
	all := h.storage.All()
	if len(all) == 0 {
		return false
	}
	for _, todo := range all {
		if todo.Status != "completed" {
			return false
		}
	}
	return true
}

func (h *todoHandler) listTodos(_ context.Context, _ tools.ToolCall) (*tools.ToolCallResult, error) {
	todos := h.storage.All()
	if todos == nil {
		todos = []Todo{}
	}
	return h.jsonResult(ListTodosOutput{Todos: todos})
}

func (t *TodoTool) Tools(context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Name:         ToolNameCreateTodo,
			Category:     "todo",
			Description:  "Create a new todo item with a description",
			Parameters:   tools.MustSchemaFor[CreateTodoArgs](),
			OutputSchema: tools.MustSchemaFor[Todo](),
			Handler:      tools.NewHandler(t.handler.createTodo),
			Annotations: tools.ToolAnnotations{
				Title:        "Create TODO",
				ReadOnlyHint: true, // Technically not read-only but has practically no destructive side effects.
			},
		},
		{
			Name:         ToolNameCreateTodos,
			Category:     "todo",
			Description:  "Create a list of new todo items with descriptions",
			Parameters:   tools.MustSchemaFor[CreateTodosArgs](),
			OutputSchema: tools.MustSchemaFor[CreateTodosOutput](),
			Handler:      tools.NewHandler(t.handler.createTodos),
			Annotations: tools.ToolAnnotations{
				Title:        "Create TODOs",
				ReadOnlyHint: true, // Technically not read-only but has practically no destructive side effects.
			},
		},
		{
			Name:         ToolNameUpdateTodos,
			Category:     "todo",
			Description:  "Update the status of one or more todo item(s)",
			Parameters:   tools.MustSchemaFor[UpdateTodosArgs](),
			OutputSchema: tools.MustSchemaFor[UpdateTodosOutput](),
			Handler:      tools.NewHandler(t.handler.updateTodos),
			Annotations: tools.ToolAnnotations{
				Title:        "Update TODOs",
				ReadOnlyHint: true, // Technically not read-only but has practically no destructive side effects.
			},
		},
		{
			Name:         ToolNameListTodos,
			Category:     "todo",
			Description:  "List all current todos with their status",
			OutputSchema: tools.MustSchemaFor[ListTodosOutput](),
			Handler:      t.handler.listTodos,
			Annotations: tools.ToolAnnotations{
				Title:        "List TODOs",
				ReadOnlyHint: true,
			},
		},
	}, nil
}
