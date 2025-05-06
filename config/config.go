package config

import (
	"fmt"
	"os"

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

// validateConfig ensures the configuration is valid
func validateConfig(config *Config) error {
	// Validate that all models referenced in agents exist
	for agentName, agent := range config.Agents {
		if _, exists := config.Models[agent.Model]; !exists {
			return fmt.Errorf("agent '%s' references non-existent model '%s'", agentName, agent.Model)
		}

		// Validate that all sub-agents exist
		for _, subAgentName := range agent.SubAgents {
			if _, exists := config.Agents[subAgentName]; !exists {
				return fmt.Errorf("agent '%s' references non-existent sub-agent '%s'", agentName, subAgentName)
			}
		}
	}

	return nil
}

// GetAgent returns an agent configuration by name
func (c *Config) GetAgent(name string) (*Agent, error) {
	agent, exists := c.Agents[name]
	if !exists {
		return nil, fmt.Errorf("agent '%s' not found in configuration", name)
	}
	return &agent, nil
}

// GetModelConfig returns a model configuration by name
func (c *Config) GetModelConfig(name string) (*ModelConfig, error) {
	model, exists := c.Models[name]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found in configuration", name)
	}
	return &model, nil
}
