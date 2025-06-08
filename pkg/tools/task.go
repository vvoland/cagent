package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

type TaskTool struct {
}

func NewTaskTool() *TaskTool {
	return &TaskTool{}
}

func (t *TaskTool) Instructions() string {
	return ""
}

func (t *TaskTool) Tools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Function: &FunctionDefinition{
				Name: "transfer_task",
				Description: `Use this function to transfer a task to the selected team member.
            You must provide a clear and concise description of the task the member should achieve AND the expected output.
`,
				Parameters: mcp.ToolInputSchema{
					Type: "object",
					Properties: map[string]any{
						"agent": map[string]any{
							"type":        "string",
							"description": "The name of the agent to transfer the task to.",
						},
						"task": map[string]any{
							"type":        "string",
							"description": "A clear and concise description of the task the member should achieve.",
						},
					},
					Required: []string{"agent", "task"},
				},
			},
		},
	}, nil
}

func (t *TaskTool) Start(ctx context.Context) error {
	return nil
}

func (t *TaskTool) Stop() error {
	return nil
}
