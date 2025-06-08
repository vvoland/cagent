package agent

import (
	"context"

	"github.com/rumpl/cagent/pkg/tools"
)

type toolwrapper struct {
	allTools []tools.Tool
}

func (t *toolwrapper) Tools(ctx context.Context) ([]tools.Tool, error) {
	return t.allTools, nil
}

func (t *toolwrapper) Instructions() string {
	return ""
}

func (t *toolwrapper) Start(ctx context.Context) error {
	return nil
}

func (t *toolwrapper) Stop() error {
	return nil
}
