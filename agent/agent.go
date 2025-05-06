package agent

import (
	"fmt"
	"slices"

	"github.com/rumpl/cagent/agent/tools"
	"github.com/rumpl/cagent/config"
	goOpenAI "github.com/sashabaranov/go-openai"
)

// Agent represents an AI agent with configuration and tools
type Agent struct {
	config *config.Agent
	tools  []goOpenAI.Tool
}

// NewAgent creates a new agent from configuration
func NewAgent(cfg *config.Config, agentName string, parentPath string) (*Agent, error) {
	agentConfig, err := cfg.GetAgent(agentName)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	tools, err := tools.GetToolsForAgent(cfg, agentName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tools: %w", err)
	}

	agent := &Agent{
		config: agentConfig,
		tools:  tools,
	}

	return agent, nil
}

// GetInstructions returns the agent's instructions
func (a *Agent) GetInstructions() string {
	return a.config.Instruction
}

// GetSubAgents returns the list of sub-agent names
func (a *Agent) GetSubAgents() []string {
	return a.config.SubAgents
}

// HasSubAgents checks if the agent has sub-agents
func (a *Agent) HasSubAgents() bool {
	return len(a.config.SubAgents) > 0
}

// IsSubAgent checks if a given agent name is a sub-agent
func (a *Agent) IsSubAgent(name string) bool {
	return slices.Contains(a.config.SubAgents, name)
}

// GetModel returns the model name used by the agent
func (a *Agent) GetModel() string {
	return a.config.Model
}

// GetTools returns the tools available to this agent
func (a *Agent) GetTools() []goOpenAI.Tool {
	return a.tools
}
