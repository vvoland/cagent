package team

import (
	"fmt"

	"github.com/docker/cagent/pkg/agent"
)

type Team struct {
	agents map[string]*agent.Agent
}

func New(agents ...*agent.Agent) *Team {
	agentsByName := make(map[string]*agent.Agent)
	for _, agent := range agents {
		agentsByName[agent.Name()] = agent
	}

	return &Team{agents: agentsByName}
}

func (t *Team) Get(name string) *agent.Agent {
	return t.agents[name]
}

func (t *Team) Size() int {
	return len(t.agents)
}

func (t *Team) StopToolSets() error {
	for _, agent := range t.agents {
		if err := agent.StopToolSets(); err != nil {
			return fmt.Errorf("failed to stop tool sets: %w", err)
		}
	}

	return nil
}
