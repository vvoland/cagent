package team

import "github.com/docker/cagent/pkg/agent"

type Team struct {
	agents map[string]*agent.Agent
}

func New(agents map[string]*agent.Agent) *Team {
	return &Team{agents: agents}
}

func (t *Team) Agents() map[string]*agent.Agent {
	return t.agents
}

func (t *Team) Get(name string) *agent.Agent {
	return t.agents[name]
}

func (t *Team) Size() int {
	return len(t.agents)
}
