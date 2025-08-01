package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

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
