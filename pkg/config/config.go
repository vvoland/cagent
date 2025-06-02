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

func Agents(ctx context.Context, path string) (map[string]*agent.Agent, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	agents := make(map[string]*agent.Agent)
	for name, agentConfig := range cfg.Agents {
		opts := []agent.Opt{
			agent.WithName(name),
			agent.WithModel(agentConfig.Model),
			agent.WithDescription(agentConfig.Description),
			agent.WithAddDate(agentConfig.AddDate),
		}

		tools, err := getToolsForAgent(ctx, cfg, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get tools: %w", err)
		}

		opts = append(opts, agent.WithToolSet(tools))

		agents[name] = agent.New(name, agentConfig.Instruction, opts...)
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
func getToolsForAgent(ctx context.Context, cfg *Config, agentName string) ([]tools.ToolSet, error) {
	var t []tools.ToolSet

	// a := cfg.Agents[agentName]
	// if len(a.SubAgents) > 0 {
	t = append(t, tools.NewAgentTransferTool())
	// }

	toolDefs := cfg.Agents[agentName].Tools
	for _, toolDef := range toolDefs {
		if toolDef.Type == "mcp" {
			mcpc, err := mcp.NewToolset(ctx, toolDef.Command, toolDef.Args)
			if err != nil {
				return nil, fmt.Errorf("failed to create mcp client: %w", err)
			}

			if err := mcpc.Start(ctx); err != nil {
				return nil, fmt.Errorf("failed to start mcp client: %w", err)
			}

			t = append(t, mcpc)
		}
		if toolDef.Type == "builtin" {
			tt := tools.NewThinkTool()
			t = append(t, tt)

		}
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
