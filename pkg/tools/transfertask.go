package tools

import (
	"context"
)

type TransferTaskTool struct {
}

func NewTransferTaskTool() *TransferTaskTool {
	return &TransferTaskTool{}
}

func (t *TransferTaskTool) Instructions() string {
	return ""
}

func (t *TransferTaskTool) Tools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Function: &FunctionDefinition{
				Name: "transfer_task",
				Description: `Use this function to transfer a task to the selected team member.
            You must provide a clear and concise description of the task the member should achieve AND the expected output.
`,
				Parameters: FunctionParamaters{
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

func (t *TransferTaskTool) Start(ctx context.Context) error {
	return nil
}

func (t *TransferTaskTool) Stop() error {
	return nil
}
