package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type TodoTool struct {
	handler *todoHandler
}

type Todo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"` // "pending", "completed"
}

type todoHandler struct {
	todos map[string]Todo
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
            
            2. While working:
               - Use list_todos frequently to keep track of remaining work
               - Mark todos as "completed" when finished
            
            3. Task Management Rules:
               - Never start a new task without creating a todo for it
               - Always check list_todos before responding to ensure no steps are missed
               - Update todo status to reflect current progress
            
            This toolset is REQUIRED for maintaining task state and ensuring all steps are completed.`
}

func (h *todoHandler) createTodo(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
	var params struct {
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	id := fmt.Sprintf("todo_%d", len(h.todos)+1)
	todo := Todo{
		ID:          id,
		Description: params.Description,
		Status:      "pending",
	}
	h.todos[id] = todo

	return &ToolCallResult{
		Output: fmt.Sprintf("Created todo %s: %s", id, params.Description),
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
	var output strings.Builder
	output.WriteString("Current todos:\n")

	for _, todo := range h.todos {
		output.WriteString(fmt.Sprintf("- [%s] %s (Status: %s)\n",
			todo.ID, todo.Description, todo.Status))
	}

	return &ToolCallResult{
		Output: output.String(),
	}, nil
}

func (t *TodoTool) Tools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Function: &FunctionDefinition{
				Name:        "create_todo",
				Description: "Create a new todo item with a description",
				Parameters: FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"description": map[string]any{
							"type":        "string",
							"description": "Description of the todo item",
						},
					},
					Required: []string{"description"},
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
							"description": "New status (pending, completed)",
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
	}, nil
}

func (t *TodoTool) Start(ctx context.Context) error {
	return nil
}

func (t *TodoTool) Stop() error {
	return nil
}
