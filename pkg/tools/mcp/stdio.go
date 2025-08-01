package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/client"
)

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
