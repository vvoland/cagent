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
	agent, err := cfg.GetAgent(agentName)
	if err != nil {
		return nil, err
	}

	var tools []openai.Tool

	// Add tools based on the agent's configuration
	for _, toolName := range agent.Tools {
		switch toolName {
		case "file_system":
			// File system related tools
			tools = append(tools, getFileSystemTools()...)
		case "web_browser":
			// Web browser related tools
			tools = append(tools, getWebBrowserTools()...)
		}
	}

	tools = append(tools, AgentTransfer())

	return tools, nil
}

// getFileSystemTools returns file system related tools
func getFileSystemTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        "read_file",
				Description: "Read the contents of a file",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the file to read",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        "write_file",
				Description: "Write content to a file",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the file to write",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "Content to write to the file",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        "list_directory",
				Description: "List the contents of a directory",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the directory to list",
						},
					},
					"required": []string{"path"},
				},
			},
		},
	}
}

// getWebBrowserTools returns web browser related tools
func getWebBrowserTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        "search_web",
				Description: "Search the web for information",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Search query",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        "fetch_url",
				Description: "Fetch the contents of a URL",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{
							"type":        "string",
							"description": "URL to fetch",
						},
					},
					"required": []string{"url"},
				},
			},
		},
	}
}
