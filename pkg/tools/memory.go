package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rumpl/cagent/pkg/memory/database"
	"github.com/rumpl/cagent/pkg/memorymanager"
)

type MemoryTool struct {
	manager memorymanager.Manager
}

func NewMemoryTool(manager memorymanager.Manager) *MemoryTool {
	return &MemoryTool{
		manager: manager,
	}
}

func (t *MemoryTool) Instructions() string {
	return `## Using the memory tool

Before taking any action or responding to the user use the "get_memories" tool to remember things about the user.
Do not talk about using the tool, just use it.

## Rules
- Use the memory tool generously to remember things about the user.`
}

func (t *MemoryTool) Tools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Function: &FunctionDefinition{
				Name:        "add_memory",
				Description: "Add a new memory to the database",
				Parameters: FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"memory": map[string]any{
							"type":        "string",
							"description": "The memory content to store",
						},
					},
					Required: []string{"memory"},
				},
			},
			Handler: t.handleAddMemory,
		},
		{
			Function: &FunctionDefinition{
				Name:        "get_memories",
				Description: "Retrieve all stored memories",
				Parameters: FunctionParamaters{
					Type:       "object",
					Properties: map[string]any{},
				},
			},
			Handler: t.handleGetMemories,
		},
		{
			Function: &FunctionDefinition{
				Name:        "delete_memory",
				Description: "Delete a specific memory by ID",
				Parameters: FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "The ID of the memory to delete",
						},
					},
					Required: []string{"id"},
				},
			},
			Handler: t.handleDeleteMemory,
		},
	}, nil
}

func (t *MemoryTool) handleAddMemory(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
	var args struct {
		Memory string `json:"memory"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	memory := database.UserMemory{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		CreatedAt: time.Now().Format(time.RFC3339),
		Memory:    args.Memory,
	}

	if err := t.manager.AddMemory(ctx, memory); err != nil {
		return nil, fmt.Errorf("failed to add memory: %w", err)
	}

	return &ToolCallResult{
		Output: fmt.Sprintf("Memory added successfully with ID: %s", memory.ID),
	}, nil
}

func (t *MemoryTool) handleGetMemories(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
	memories, err := t.manager.GetMemories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories: %w", err)
	}

	result, err := json.Marshal(memories)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal memories: %w", err)
	}

	return &ToolCallResult{
		Output: string(result),
	}, nil
}

func (t *MemoryTool) handleDeleteMemory(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	memory := database.UserMemory{
		ID: args.ID,
	}

	if err := t.manager.DeleteMemory(ctx, memory); err != nil {
		return nil, fmt.Errorf("failed to delete memory: %w", err)
	}

	return &ToolCallResult{
		Output: fmt.Sprintf("Memory with ID %s deleted successfully", args.ID),
	}, nil
}

func (t *MemoryTool) Start(ctx context.Context) error {
	return nil
}

func (t *MemoryTool) Stop() error {
	return nil
}
