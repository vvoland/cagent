package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
			agent.WithAddDate(agentConfig.AddDate),
		}

		tools, err := getToolsForAgent(cfg, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get tools: %w", err)
		}

		opts = append(opts, agent.WithTools(tools))

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
func getToolsForAgent(cfg *Config, agentName string) ([]tools.Tool, error) {
	var t []tools.Tool

	a := cfg.Agents[agentName]
	if len(a.SubAgents) > 0 {
		t = append(t, tools.AgentTransfer())
	}
	toolDefs := cfg.Agents[agentName].Tools
	for _, toolDef := range toolDefs {
		mcpc, err := mcp.New(context.Background(), toolDef.Command, toolDef.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to create mcp client: %w", err)
		}

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

// LoadAgentsFromDirectory loads all agent configurations from a directory
func LoadAgentsFromDirectory(dir string) (map[string]*agent.Agent, error) {
	if dir == "" {
		return nil, fmt.Errorf("directory path is required")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	agents := make(map[string]*agent.Agent)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			configPath := filepath.Join(dir, entry.Name())
			dirAgents, err := Agents(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load agents from %s: %w", configPath, err)
			}

			// Merge agents from this file into the main map
			for name, agent := range dirAgents {
				if _, exists := agents[name]; exists {
					return nil, fmt.Errorf("duplicate agent name '%s' found in %s", name, configPath)
				}
				agents[name] = agent
			}
		}
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agent configurations found in directory %s", dir)
	}

	return agents, nil
}
