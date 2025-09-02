package agent

import (
	"context"

	"github.com/docker/cagent/pkg/tools"
)

type toolwrapper struct {
	allTools []tools.Tool
}

func (t *toolwrapper) Tools(context.Context) ([]tools.Tool, error) {
	return t.allTools, nil
}

func (t *toolwrapper) Instructions() string {
	return ""
}

func (t *toolwrapper) Start(context.Context) error {
	return nil
}

func (t *toolwrapper) Stop() error {
	return nil
}
