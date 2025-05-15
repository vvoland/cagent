package agent

import (
	"slices"

	"github.com/sashabaranov/go-openai"
)

// Agent represents an AI agent
type Agent struct {
	name        string
	description string
	instruction string
	tools       []openai.Tool
	model       string
	subAgents   []*Agent
	parents     []*Agent
}

type AgentOpt func(a *Agent)

func WithInstruction(prompt string) AgentOpt {
	return func(a *Agent) {
		a.instruction = prompt
	}
}

func WithTools(tools []openai.Tool) AgentOpt {
	return func(a *Agent) {
		a.tools = tools
	}
}

func WithDescription(description string) AgentOpt {
	return func(a *Agent) {
		a.description = description
	}
}

func WithName(name string) AgentOpt {
	return func(a *Agent) {
		a.name = name
	}
}

func WithModel(model string) AgentOpt {
	return func(a *Agent) {
		a.model = model
	}
}

func WithSubAgents(subAgents []*Agent) AgentOpt {
	return func(a *Agent) {
		a.subAgents = subAgents
		for _, subAgent := range subAgents {
			subAgent.parents = append(subAgent.parents, a)
		}
	}
}

// New creates a new agent
func New(agentName string, prompt string, opts ...AgentOpt) (*Agent, error) {
	// tools, err := tools.GetToolsForAgent(cfg, agentName)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get tools: %w", err)
	// }

	agent := &Agent{
		// tools:       tools,
		instruction: prompt,
	}

	for _, opt := range opts {
		opt(agent)
	}

	return agent, nil
}

func (a *Agent) Name() string {
	return a.name
}

// Instruction returns the agent's instructions
func (a *Agent) Instruction() string {
	return a.instruction
}

// Description returns the agent's description
func (a *Agent) Description() string {
	return a.description
}

// SubAgents returns the list of sub-agent names
func (a *Agent) SubAgents() []*Agent {
	return a.subAgents
}

// HasSubAgents checks if the agent has sub-agents
func (a *Agent) HasSubAgents() bool {
	return len(a.subAgents) > 0
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
func (a *Agent) Model() string {
	return a.model
}

// Tools returns the tools available to this agent
func (a *Agent) Tools() []openai.Tool {
	return a.tools
}
