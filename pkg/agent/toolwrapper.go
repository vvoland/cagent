package agent

import (
	"context"

	"github.com/docker/cagent/pkg/tools"
)

type toolWrapper struct {
	allTools []tools.Tool
}

func (t *toolWrapper) Tools(context.Context) ([]tools.Tool, error) {
	return t.allTools, nil
}

func (t *toolWrapper) Instructions() string {
	return ""
}

func (t *toolWrapper) Start(context.Context) error {
	return nil
}

func (t *toolWrapper) Stop() error {
	return nil
}
