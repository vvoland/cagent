package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"

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

func Agents(ctx context.Context, path string, logger *slog.Logger) (map[string]*agent.Agent, error) {
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

		agentTools, err := getToolsForAgent(ctx, cfg, name, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to get tools: %w", err)
		}

		opts = append(opts, agent.WithToolSet(agentTools))

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

func hasParents(cfg *Config, agentName string) bool {
	for _, agent := range cfg.Agents {
		if len(agent.SubAgents) > 0 {
			if slices.Contains(agent.SubAgents, agentName) {
				return true
			}
		}
	}

	return false
}

// getToolsForAgent returns the tool definitions for an agent based on its configuration
func getToolsForAgent(ctx context.Context, cfg *Config, agentName string, logger *slog.Logger) ([]tools.ToolSet, error) {
	a, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found in configuration", agentName)
	}

	var t []tools.ToolSet

	if hasParents(cfg, agentName) || len(a.SubAgents) > 0 {
		t = append(t, tools.NewAgentTransferTool())
	}

	if a.Think {
		t = append(t, tools.NewThinkTool())
	}

	toolsets := a.Toolsets
	for _, toolset := range toolsets {
		if toolset.Type == "builtin" {
			t = append(t, tools.NewBashTool())
		}
		if toolset.Type != "mcp" {
			continue
		}

		// Convert env map to string slice
		envSlice := make([]string, 0, len(toolset.Env))
		for k, v := range toolset.Env {
			envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
		}
		mcpc, err := mcp.NewToolset(ctx, toolset.Command, toolset.Args, envSlice, toolset.Tools, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create mcp client: %w", err)
		}

		if err := mcpc.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start mcp client: %w", err)
		}

		t = append(t, mcpc)
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
