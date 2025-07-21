package config

import (
	"fmt"
	"os"
	"strings"

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

	for _, model := range config.Models {
		if model.ParallelToolCalls == nil {
			model.ParallelToolCalls = new(bool)
			*model.ParallelToolCalls = true
		}
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig ensures the configuration is valid
func validateConfig(cfg *Config) error {
	// Validate that all models referenced in agents exist
	for agentName := range cfg.Agents {
		agent := cfg.Agents[agentName]
		modelNames := strings.SplitSeq(agent.Model, ",")
		for modelName := range modelNames {
			if _, exists := cfg.Models[modelName]; !exists {
				return fmt.Errorf("agent '%s' references non-existent model '%s'", agentName, modelName)
			}
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
