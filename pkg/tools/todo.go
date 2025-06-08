package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type TodoTool struct {
	handler *todoHandler
}

type Todo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`   // "pending", "in_progress", "completed"
	Priority    string `json:"priority"` // "high", "medium", "low"
}

type todoHandler struct {
	todos map[string]Todo
	mu    sync.RWMutex
}

func NewTodoTool() *TodoTool {
	return &TodoTool{
		handler: &todoHandler{
			todos: make(map[string]Todo),
		},
	}
}

func (t *TodoTool) Instructions() string {
	return `## Using the Todo Tools
            IMPORTANT: You MUST use these tools to track the progress of your tasks:
            
            1. Before starting any complex task:
               - Create a todo for each major step using create_todo
               - Break down complex steps into smaller todos
               - Set appropriate priorities
            
            2. While working:
               - Use update_todo to mark tasks as "in_progress" when starting them
               - Use list_todos frequently to keep track of remaining work
               - Mark todos as "completed" when finished
            
            3. Task Management Rules:
               - Never start a new task without creating a todo for it
               - Always check list_todos before responding to ensure no steps are missed
               - Update todo status to reflect current progress
               - Remove todos only when they are obsolete or no longer relevant
            
            This toolset is REQUIRED for maintaining task state and ensuring all steps are completed.`
}



func (h *todoHandler) createTodo(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
	var params struct {
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	id := fmt.Sprintf("todo_%d", len(h.todos)+1)
	todo := Todo{
		ID:          id,
		Description: params.Description,
		Status:      "pending",
		Priority:    params.Priority,
	}
	h.todos[id] = todo

	return &ToolCallResult{
		Output: fmt.Sprintf("Created todo %s: %s (Priority: %s)", id, params.Description, params.Priority),
	}, nil
}

func (h *todoHandler) updateTodo(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
	var params struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	todo, exists := h.todos[params.ID]
	if !exists {
		return nil, fmt.Errorf("todo %s not found", params.ID)
	}

	todo.Status = params.Status
	h.todos[params.ID] = todo

	return &ToolCallResult{
		Output: fmt.Sprintf("Updated todo %s status to: %s", params.ID, params.Status),
	}, nil
}

func (h *todoHandler) listTodos(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var output strings.Builder
	output.WriteString("Current todos:\n")

	for _, todo := range h.todos {
		output.WriteString(fmt.Sprintf("- [%s] %s (Priority: %s, Status: %s)\n",
			todo.ID, todo.Description, todo.Priority, todo.Status))
	}

	return &ToolCallResult{
		Output: output.String(),
	}, nil
}

func (h *todoHandler) removeTodo(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
	var params struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.todos[params.ID]; !exists {
		return nil, fmt.Errorf("todo %s not found", params.ID)
	}

	delete(h.todos, params.ID)

	return &ToolCallResult{
		Output: fmt.Sprintf("Removed todo %s", params.ID),
	}, nil
}

func (t *TodoTool) Tools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Function: &FunctionDefinition{
				Name:        "create_todo",
				Description: "Create a new todo item with a description and priority",
				Parameters: FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"description": map[string]any{
							"type":        "string",
							"description": "Description of the todo item",
						},
						"priority": map[string]any{
							"type":        "string",
							"description": "Priority level (high, medium, low)",
						},
					},
					Required: []string{"description", "priority"},
				},
			},
			Handler: t.handler.createTodo,
		},
		{
			Function: &FunctionDefinition{
				Name:        "update_todo",
				Description: "Update the status of a todo item",
				Parameters: FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "ID of the todo item",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "New status (pending, in_progress, completed)",
						},
					},
					Required: []string{"id", "status"},
				},
			},
			Handler: t.handler.updateTodo,
		},
		{
			Function: &FunctionDefinition{
				Name:        "list_todos",
				Description: "List all current todos with their status",
			},
			Handler: t.handler.listTodos,
		},
		{
			Function: &FunctionDefinition{
				Name:        "remove_todo",
				Description: "Remove a todo item by ID",
				Parameters: FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "ID of the todo item to remove",
						},
					},
					Required: []string{"id"},
				},
			},
			Handler: t.handler.removeTodo,
		},
	}, nil
}

func (t *TodoTool) Start(ctx context.Context) error {
	return nil
}

func (t *TodoTool) Stop() error {
	return nil
}
