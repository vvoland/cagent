package mcp

import (
	"context"
	"fmt"

	"github.com/rumpl/cagent/pkg/tools"
)

// Toolset represents a set of MCP tools
type Toolset struct {
	c          *Client
	toolFilter []string
}

// NewToolset creates a new MCP toolset
func NewToolset(ctx context.Context, command string, args, env, toolFilter []string) (*Toolset, error) {
	mcpc, err := New(ctx, command, args, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp client: %w", err)
	}

	return &Toolset{
		c:          mcpc,
		toolFilter: toolFilter,
	}, nil
}

// Handler returns the tool handler
func (t *Toolset) Handler() tools.ToolHandler {
	return t.c
}

// Instructions returns the toolset instructions
func (t *Toolset) Instructions() string {
	return ""
}

// Tools returns the available tools
func (t *Toolset) Tools(ctx context.Context) ([]tools.Tool, error) {
	return t.c.ListTools(ctx, t.toolFilter)
}

// Start starts the toolset
func (t *Toolset) Start(ctx context.Context) error {
	return t.c.Start(ctx)
}

// Stop stops the toolset
func (t *Toolset) Stop() error {
	return t.c.Stop()
}
