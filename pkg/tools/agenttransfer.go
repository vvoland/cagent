package tools

import (
	"context"
)

type AgentTransferTool struct {
}

func NewAgentTransferTool() *AgentTransferTool {
	return &AgentTransferTool{}
}

func (t *AgentTransferTool) Instructions() string {
	return ""
}

func (t *AgentTransferTool) Tools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Function: &FunctionDefinition{
				Name:        "transfer_to_agent",
				Description: "Transfer the question to another agent",
				Parameters: FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"agent": map[string]any{
							"type":        "string",
							"description": "The name of the agent to transfer the question to",
						},
					},
					Required: []string{"agent"},
				},
			},
		},
	}, nil
}

func (t *AgentTransferTool) Start(ctx context.Context) error {
	return nil
}

func (t *AgentTransferTool) Stop() error {
	return nil
}
