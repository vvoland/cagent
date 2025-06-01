package mcp

import (
	"context"
	"fmt"

	"github.com/rumpl/cagent/pkg/tools"
)

type MCPToolset struct {
	c *Client
}

func NewMcpToolset(ctx context.Context, command string, args []string) (*MCPToolset, error) {
	mcpc, err := New(ctx, command, args)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp client: %w", err)
	}

	return &MCPToolset{
		c: mcpc,
	}, nil
}

func (t *MCPToolset) Handler() tools.ToolHandler {
	return t.c
}

func (t *MCPToolset) Start(ctx context.Context) error {
	return t.c.Start(ctx)
}

func (t *MCPToolset) Stop() error {
	return t.c.Stop()
}

func (t *MCPToolset) Tools(ctx context.Context) ([]tools.Tool, error) {
	return t.c.ListTools(ctx)
}
