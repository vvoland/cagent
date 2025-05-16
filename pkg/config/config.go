package config

import (
	"context"
	"fmt"
	"os"

	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/mcp"
	"github.com/rumpl/cagent/pkg/tools"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func Agents(path string) (map[string]*agent.Agent, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	agents := make(map[string]*agent.Agent)
	for name, agentConfig := range cfg.Agents {
		opts := []agent.AgentOpt{
			agent.WithName(name),
			agent.WithModel(agentConfig.Model),
			agent.WithDescription(agentConfig.Description),
		}

		tools, err := getToolsForAgent(cfg, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get tools: %w", err)
		}

		opts = append(opts, agent.WithTools(tools))

		a, err := agent.New(name, agentConfig.Instruction, opts...)
		if err != nil {
			continue
		}

		agents[name] = a
	}

	for name, agentConfig := range cfg.Agents {
		if len(agentConfig.SubAgents) > 0 {
			subAgents := make([]*agent.Agent, 0, len(agentConfig.SubAgents))
			for _, subName := range agentConfig.SubAgents {
				if subAgent, exists := agents[subName]; exists {
					subAgents = append(subAgents, subAgent)
				}
			}

			if a, exists := agents[name]; exists && len(subAgents) > 0 {
				agent.WithSubAgents(subAgents)(a)
			}
		}
	}

	return agents, nil
}

// getToolsForAgent returns the tool definitions for an agent based on its configuration
func getToolsForAgent(cfg *Config, agentName string) ([]tools.Tool, error) {
	var t []tools.Tool

	t = append(t, tools.AgentTransfer())

	toolDefs := cfg.Agents[agentName].Tools
	for _, toolDef := range toolDefs {
		mcpc, _ := mcp.New(context.Background(), toolDef.Command, toolDef.Args)
		mcpc.Start(context.Background())
		tools, _ := mcpc.ListTools(context.Background())
		t = append(t, tools...)
	}

	return t, nil
}

// validateConfig ensures the configuration is valid
func validateConfig(cfg *Config) error {
	// Validate that all models referenced in agents exist
	for agentName, agent := range cfg.Agents {
		if _, exists := cfg.Models[agent.Model]; !exists {
			return fmt.Errorf("agent '%s' references non-existent model '%s'", agentName, agent.Model)
		}

		// Validate that all sub-agents exist
		for _, subAgentName := range agent.SubAgents {
			if _, exists := cfg.Agents[subAgentName]; !exists {
				return fmt.Errorf("agent '%s' references non-existent sub-agent '%s'", agentName, subAgentName)
			}
		}
	}

	return nil
}

// GetModelConfig returns a model configuration by name
func (c *Config) GetModelConfig(name string) (*ModelConfig, error) {
	model, exists := c.Models[name]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found in configuration", name)
	}
	return &model, nil
}
