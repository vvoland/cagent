package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rumpl/cagent/pkg/tools"
)

// Client implements an MCP client for interacting with MCP servers
type Client struct {
	client *client.Client
	tools  []tools.Tool
}

// New creates a new MCP client that can start an stdio MCP server
func New(ctx context.Context, command string, args, env []string) (*Client, error) {
	mcpClient, err := client.NewStdioMCPClient(command, env, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create stdio client: %w", err)
	}

	return &Client{
		client: mcpClient,
		tools:  []tools.Tool{},
	}, nil
}

// Start initializes and starts the MCP server connection
func (c *Client) Start(ctx context.Context) error {
	if err := c.client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP client: %w", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "cagent",
		Version: "1.0.0",
	}

	_, err := c.client.Initialize(ctx, initRequest)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	return nil
}

// Stop stops the MCP server
func (c *Client) Stop() error {
	return c.client.Close()
}

// ListTools fetches available tools from the MCP server
func (c *Client) ListTools(ctx context.Context) ([]tools.Tool, error) {
	resp, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	var toolsList []tools.Tool
	for i := range resp.Tools {
		t := &resp.Tools[i]
		tool := tools.Tool{
			Type: "function",
		}

		tool.Function = &tools.FunctionDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		}

		tool.Handler = c

		toolsList = append(toolsList, tool)
	}

	c.tools = toolsList
	return toolsList, nil
}

// CallTool calls a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	request := mcp.CallToolRequest{}
	request.Params.Name = toolCall.Function.Name
	request.Params.Arguments = args
	if request.Params.Arguments == nil {
		request.Params.Arguments = map[string]any{}
	}
	// MCP servers return an error if the args are empty so we make sure
	// there is at least one argument
	// if len(request.Params.Arguments) == 0 {
	// 	request.Params.Arguments["args"] = "..."
	// }

	resp, err := c.client.CallTool(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	return processMCPContent(resp), nil
}

func processMCPContent(toolResult *mcp.CallToolResult) *tools.ToolCallResult {
	finalContent := ""
	for _, resultContent := range toolResult.Content {
		if textContent, ok := resultContent.(mcp.TextContent); ok {
			finalContent += textContent.Text
		}
	}

	return &tools.ToolCallResult{
		Output: finalContent,
	}
}

// GetToolByName returns a tool by name
func (c *Client) GetToolByName(name string) (tools.Tool, error) {
	for _, tool := range c.tools {
		if tool.Function != nil && tool.Function.Name == name {
			return tool, nil
		}
	}
	return tools.Tool{}, fmt.Errorf("tool %s not found", name)
}

// CallToolWithArgs is a convenience method to call a tool with arguments
func (c *Client) CallToolWithArgs(ctx context.Context, toolName string, args interface{}) (*tools.ToolCallResult, error) {
	argsBytes, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	toolCall := tools.ToolCall{
		Type: "function",
		Function: tools.FunctionCall{
			Name:      toolName,
			Arguments: string(argsBytes),
		},
	}

	return c.CallTool(ctx, toolCall)
}
