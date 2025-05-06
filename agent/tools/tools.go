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

// ReadFile creates a tool definition for reading a file's content
func ReadFile() openai.Tool {
	return openai.Tool{
		Type: "function",
		Function: &openai.FunctionDefinition{
			Name:        "read_file",
			Description: "Reads the content of a file and returns it as a string",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "The path to the file to read",
					},
				},
				"required": []string{"file_path"},
			},
		},
	}
}

// WriteFile creates a tool definition for writing content to a file
func WriteFile() openai.Tool {
	return openai.Tool{
		Type: "function",
		Function: &openai.FunctionDefinition{
			Name:        "write_file",
			Description: "Writes content to a file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "The path to the file to write to",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "The content to write to the file",
					},
				},
				"required": []string{"file_path", "content"},
			},
		},
	}
}

// BuildDockerfile creates a tool definition for building a Dockerfile for a directory
func BuildDockerfile() openai.Tool {
	return openai.Tool{
		Type: "function",
		Function: &openai.FunctionDefinition{
			Name:        "build_dockerfile",
			Description: "Builds a Dockerfile for the given directory",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"directory_path": map[string]any{
						"type":        "string",
						"description": "The path to the directory to build a Dockerfile for",
					},
				},
				"required": []string{"directory_path"},
			},
		},
	}
}

// GetToolsForAgent returns the tool definitions for an agent based on its configuration
func GetToolsForAgent(cfg *config.Config, agentName string) ([]openai.Tool, error) {
	var tools []openai.Tool

	tools = append(tools, AgentTransfer())

	// Add agent-specific tools based on configuration
	agent, exists := cfg.Agents[agentName]
	if exists {
		for _, tool := range agent.Tools {
			switch tool {
			case "read_file":
				tools = append(tools, ReadFile())
			case "write_file":
				tools = append(tools, WriteFile())
			case "build_dockerfile":
				tools = append(tools, BuildDockerfile())
			}
		}
	}

	return tools, nil
}
