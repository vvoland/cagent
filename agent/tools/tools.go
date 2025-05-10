package tools

import (
	"github.com/rumpl/cagent/config"
	"github.com/sashabaranov/go-openai"
)

// AgentTransfer creates a tool definition for transferring control to another agent
func AgentTransfer() openai.Tool {
	return openai.Tool{
		Type: "function",
		Function: &openai.FunctionDefinition{
			Name:        "transfer_to_agent",
			Description: "Transfer the conversation to another agent",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agent": map[string]any{
						"type":        "string",
						"description": "The name of the agent to transfer to",
					},
				},
				"required": []string{"agent"},
			},
		},
	}
}

// GetToolsForAgent returns the tool definitions for an agent based on its configuration
func GetToolsForAgent(cfg *config.Config, agentName string) ([]openai.Tool, error) {
	var tools []openai.Tool

	tools = append(tools, AgentTransfer())

	return tools, nil
}
