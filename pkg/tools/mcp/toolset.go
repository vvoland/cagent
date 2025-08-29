package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/cagent/pkg/tools"
)

// Toolset represents a set of MCP tools
type Toolset struct {
	c          *Client
	toolFilter []string
	logger     *slog.Logger
}

// Make sure the MCP Toolset always implements _our_ ToolSet interface
var _ tools.ToolSet = (*Toolset)(nil)

// NewToolsetCommand creates a new MCP toolset from a command.
func NewToolsetCommand(ctx context.Context, command string, args, env, toolFilter []string, logger *slog.Logger) (*Toolset, error) {
	logger.Debug("Creating MCP toolset", "command", command, "args", args, "toolFilter", toolFilter)

	mcpc, err := NewStdioClient(ctx, command, args, env, logger)
	if err != nil {
		logger.Error("Failed to create MCP client", "error", err)
		return nil, fmt.Errorf("failed to create mcp client: %w", err)
	}

	return &Toolset{
		c:          mcpc,
		toolFilter: toolFilter,
		logger:     logger,
	}, nil
}

// NewToolsetRemote creates a new MCP toolset from a remote MCP Server.
func NewToolsetRemote(ctx context.Context, url, transport string, headers map[string]string, toolFilter []string, logger *slog.Logger) (*Toolset, error) {
	logger.Debug("Creating MCP toolset", "url", url, "transport", transport, "headers", headers, "toolFilter", toolFilter)

	mcpc, err := NewRemoteClient(ctx, url, transport, headers, logger)
	if err != nil {
		logger.Error("Failed to create remote MCP client", "error", err)
		return nil, fmt.Errorf("failed to create remote mcp client: %w", err)
	}

	return &Toolset{
		c:          mcpc,
		toolFilter: toolFilter,
		logger:     logger,
	}, nil
}

// Instructions returns the toolset instructions
func (t *Toolset) Instructions() string {
	return ""
}

// Tools returns the available tools
func (t *Toolset) Tools(ctx context.Context) ([]tools.Tool, error) {
	t.logger.Debug("Listing MCP tools", "toolFilter", t.toolFilter)
	mcpTools, err := t.c.ListTools(ctx, t.toolFilter)
	if err != nil {
		if ctx.Err() == context.Canceled {
			// Log at debug level on cancellation
			t.logger.Debug("MCP tools listing canceled by context")
			return nil, err
		}
		t.logger.Error("Failed to list MCP tools", "error", err)
		return nil, err
	}
	t.logger.Debug("Listed MCP tools", "count", len(mcpTools))
	return mcpTools, nil
}

// Start starts the toolset
func (t *Toolset) Start(ctx context.Context) error {
	t.logger.Debug("Starting MCP toolset")
	err := t.c.Start(ctx)
	if err != nil {
		t.logger.Error("Failed to start MCP toolset", "error", err)
		return err
	}
	t.logger.Debug("Started MCP toolset successfully")
	return nil
}

// Stop stops the toolset
func (t *Toolset) Stop() error {
	t.logger.Debug("Stopping MCP toolset")
	err := t.c.Stop()
	if err != nil {
		t.logger.Error("Failed to stop MCP toolset", "error", err)
		return err
	}
	t.logger.Debug("Stopped MCP toolset successfully")
	return nil
}
