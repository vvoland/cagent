package mcp

import (
	"context"
	"fmt"

	"github.com/rumpl/cagent/pkg/tools"
)

type Toolset struct {
	c *Client
}

func NewToolset(ctx context.Context, command string, args []string) (*Toolset, error) {
	mcpc, err := New(ctx, command, args)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp client: %w", err)
	}

	return &Toolset{
		c: mcpc,
	}, nil
}

func (t *Toolset) Handler() tools.ToolHandler {
	return t.c
}

func (t *Toolset) Instructions() string {
	return ""
}

func (t *Toolset) Start(ctx context.Context) error {
	return t.c.Start(ctx)
}

func (t *Toolset) Stop() error {
	return t.c.Stop()
}

func (t *Toolset) Tools(ctx context.Context) ([]tools.Tool, error) {
	return t.c.ListTools(ctx)
}
