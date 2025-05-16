package tools

import (
	"github.com/sashabaranov/go-openai"
)

// AgentTransfer creates a tool definition for transferring control to another agent
func AgentTransfer() openai.Tool {
	return openai.Tool{
		Type: "function",
		Function: &openai.FunctionDefinition{
			Name:        "transfer_to_agent",
			Description: "Transfer the question to another agent",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agent": map[string]any{
						"type": "string",
					},
				},
				"required": []string{"agent"},
			},
		},
	}
}
