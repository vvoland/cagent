package team

import (
	"context"
	"fmt"

	"github.com/docker/cagent/pkg/agent"
)

type Team struct {
	agents          map[string]*agent.Agent
	toolsetsStarted bool
}

func New(agents map[string]*agent.Agent) *Team {
	return &Team{agents: agents}
}

func (t *Team) Get(name string) *agent.Agent {
	return t.agents[name]
}

func (t *Team) Size() int {
	return len(t.agents)
}

func (t *Team) StartToolSets(ctx context.Context) error {
	if t.toolsetsStarted {
		return nil
	}

	for _, agent := range t.agents {
		if err := agent.StartToolSets(ctx); err != nil {
			return fmt.Errorf("failed to start tool sets: %w", err)
		}
	}

	t.toolsetsStarted = true
	return nil
}

func (t *Team) StopToolSets() error {
	if !t.toolsetsStarted {
		return nil
	}

	for _, agent := range t.agents {
		if err := agent.StopToolSets(); err != nil {
			return fmt.Errorf("failed to stop tool sets: %w", err)
		}
	}

	t.toolsetsStarted = false
	return nil
}
