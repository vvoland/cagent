package tools

import "github.com/mark3labs/mcp-go/mcp"

// AgentTransfer creates a tool definition for transferring control to another agent
func AgentTransfer() Tool {
	return Tool{
		Type: "function",
		Function: &FunctionDefinition{
			Name:        "transfer_to_agent",
			Description: "Transfer the question to another agent",
			Parameters: mcp.ToolInputSchema{
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
	}
}
