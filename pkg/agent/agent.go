package agent

import (
	"context"
	"fmt"
	"slices"

	"github.com/rumpl/cagent/pkg/memorymanager"
	"github.com/rumpl/cagent/pkg/model/provider"
	"github.com/rumpl/cagent/pkg/tools"
)

// Agent represents an AI agent
type Agent struct {
	name          string
	description   string
	instruction   string
	toolsets      []tools.ToolSet
	model         provider.Provider
	subAgents     []*Agent
	parents       []*Agent
	addDate       bool
	toolwrapper   toolwrapper
	memoryManager memorymanager.Manager
}

// New creates a new agent
func New(agentName, prompt string, opts ...Opt) *Agent {
	agent := &Agent{
		name:        agentName,
		instruction: prompt,
	}

	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

func (a *Agent) Name() string {
	return a.name
}

// Instruction returns the agent's instructions
func (a *Agent) Instruction() string {
	return a.instruction
}

func (a *Agent) AddDate() bool {
	return a.addDate
}

// Description returns the agent's description
func (a *Agent) Description() string {
	return a.description
}

// SubAgents returns the list of sub-agent names
func (a *Agent) SubAgents() []*Agent {
	return a.subAgents
}

// Parents returns the list of parent agent names
func (a *Agent) Parents() []*Agent {
	return a.parents
}

// HasSubAgents checks if the agent has sub-agents
func (a *Agent) HasSubAgents() bool {
	return len(a.subAgents) > 0
}

func (a *Agent) HasParents() bool {
	return len(a.parents) > 0
}

// IsSubAgent checks if a given agent name is a sub-agent
func (a *Agent) IsSubAgent(name string) bool {
	return slices.ContainsFunc(a.subAgents, func(s *Agent) bool {
		return s.name == name
	})
}

func (a *Agent) IsParent(name string) bool {
	return slices.ContainsFunc(a.parents, func(p *Agent) bool {
		return p.name == name
	})
}

// Model returns the model name used by the agent
func (a *Agent) Model() provider.Provider {
	return a.model
}

// Tools returns the tools available to this agent
func (a *Agent) Tools() ([]tools.Tool, error) {
	agentTools := []tools.Tool{}
	for _, toolSet := range a.toolsets {
		ta, err := toolSet.Tools(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to get tools: %w", err)
		}
		agentTools = append(agentTools, ta...)
	}

	agentTools = append(agentTools, a.toolwrapper.allTools...)

	return agentTools, nil
}

func (a *Agent) ToolImpls() []tools.ToolSet {
	return a.toolsets
}
