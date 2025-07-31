package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/docker/cagent/pkg/tools"
)

// Client implements an MCP client for interacting with MCP servers
type Client struct {
	client  *client.Client
	tools   []tools.Tool
	logger  *slog.Logger
	logType string
	logId   string
}

// NewStdioClient creates a new MCP client that can start an stdio MCP server
func NewStdioClient(ctx context.Context, command string, args, env []string, logger *slog.Logger) (*Client, error) {
	logger.Debug("Creating stdio MCP client", "command", command, "args", args)

	c, err := client.NewStdioMCPClient(command, env, args...)
	if err != nil {
		logger.Error("Failed to create stdio MCP client", "error", err)
		return nil, fmt.Errorf("failed to create stdio MCP client: %w", err)
	}

	logger.Debug("Created stdio MCP client successfully")
	return &Client{
		client:  c,
		logger:  logger,
		logType: "command",
		logId:   command,
	}, nil
}

// NewRemoteClient creates a new MCP client that can connect to a remote MCP server
func NewRemoteClient(ctx context.Context, url, transportType string, headers map[string]string, logger *slog.Logger) (*Client, error) {
	logger.Debug("Creating remote MCP client", "url", url, "transport", transportType, "headers", headers)

	var c *client.Client
	if transportType == "sse" {
		var err error
		c, err = client.NewSSEMCPClient(url, client.WithHeaders(headers))
		if err != nil {
			logger.Error("Failed to create sse remote MCP client", "error", err)
			return nil, fmt.Errorf("failed to create sse remote MCP client: %w", err)
		}
	} else {
		var err error
		c, err = client.NewStreamableHttpClient(url, transport.WithHTTPHeaders(headers))
		if err != nil {
			logger.Error("Failed to create streamable remote MCP client", "error", err)
			return nil, fmt.Errorf("failed to create streamable remote MCP client: %w", err)
		}
	}

	logger.Debug("Created remote MCP client successfully")
	return &Client{
		client:  c,
		logger:  logger,
		logType: "remote",
		logId:   url,
	}, nil
}

// Start initializes and starts the MCP server connection
func (c *Client) Start(ctx context.Context) error {
	c.logger.Debug("Starting MCP client", c.logType, c.logId)

	if err := c.client.Start(ctx); err != nil {
		c.logger.Error("Failed to start MCP client", "error", err)
		return fmt.Errorf("failed to start MCP client: %w", err)
	}

	c.logger.Debug("Initializing MCP client", c.logType, c.logId)
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "cagent",
		Version: "1.0.0",
	}

	_, err := c.client.Initialize(ctx, initRequest)
	if err != nil {
		c.logger.Error("Failed to initialize MCP client", "error", err)
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	c.logger.Debug("MCP client started and initialized successfully")
	return nil
}

// Stop stops the MCP server
func (c *Client) Stop() error {
	c.logger.Debug("Stopping MCP client")
	err := c.client.Close()
	if err != nil {
		c.logger.Error("Failed to stop MCP client", "error", err)
		return err
	}
	c.logger.Debug("MCP client stopped successfully")
	return nil
}

// ListTools fetches available tools from the MCP server
func (c *Client) ListTools(ctx context.Context, toolFilter []string) ([]tools.Tool, error) {
	c.logger.Debug("Listing tools from MCP server", "toolFilter", toolFilter)

	resp, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		c.logger.Error("Failed to list tools from MCP server", "error", err)
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	c.logger.Debug("Received tools from MCP server", "count", len(resp.Tools))

	var toolsList []tools.Tool
	for i := range resp.Tools {
		t := &resp.Tools[i]
		// If toolFilter is not empty, only include tools that are in the filter
		if len(toolFilter) > 0 && !slices.Contains(toolFilter, t.Name) {
			c.logger.Debug("Filtering out tool", "tool", t.Name)
			continue
		}

		tool := tools.Tool{
			Handler: c.CallTool,
			Function: &tools.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters: tools.FunctionParamaters{
					Type:       t.InputSchema.Type,
					Properties: t.InputSchema.Properties,
					Required:   t.InputSchema.Required,
				},
				Annotations: tools.ToolAnnotation{
					Title:           t.Annotations.Title,
					ReadOnlyHint:    t.Annotations.ReadOnlyHint,
					DestructiveHint: t.Annotations.DestructiveHint,
					IdempotentHint:  t.Annotations.IdempotentHint,
					OpenWorldHint:   t.Annotations.OpenWorldHint,
				},
			},
		}
		toolsList = append(toolsList, tool)

		c.logger.Debug("Added MCP tool", "tool", t.Name)
	}

	c.tools = toolsList
	c.logger.Debug("Finished listing MCP tools", "filtered_count", len(toolsList))
	return toolsList, nil
}

// CallTool calls a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	c.logger.Debug("Calling MCP tool", "tool", toolCall.Function.Name, "arguments", toolCall.Function.Arguments)

	if toolCall.Function.Arguments == "" {
		toolCall.Function.Arguments = "{}"
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		c.logger.Error("Failed to parse tool arguments", "tool", toolCall.Function.Name, "error", err)
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	request := mcp.CallToolRequest{}
	request.Params.Name = toolCall.Function.Name
	request.Params.Arguments = args

	resp, err := c.client.CallTool(ctx, request)
	if err != nil {
		c.logger.Error("Failed to call MCP tool", "tool", toolCall.Function.Name, "error", err)
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	result := processMCPContent(resp)
	c.logger.Debug("MCP tool call completed", "tool", toolCall.Function.Name, "output_length", len(result.Output))
	c.logger.Debug(result.Output)
	return result, nil
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
func (c *Client) CallToolWithArgs(ctx context.Context, toolName string, args any) (*tools.ToolCallResult, error) {
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
