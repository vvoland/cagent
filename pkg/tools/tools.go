package tools

// AgentTransfer creates a tool definition for transferring control to another agent
func AgentTransfer() Tool {
	return Tool{
		Type: "function",
		Function: &FunctionDefinition{
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
