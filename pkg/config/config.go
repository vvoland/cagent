package config

import (
	"fmt"
	"os"
	"strings"

	v0 "github.com/docker/cagent/pkg/config/v0"
	latest "github.com/docker/cagent/pkg/config/v1"
	v1 "github.com/docker/cagent/pkg/config/v1"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*latest.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	var config latest.Config

	switch raw["version"] {
	case nil, "0", 0:
		var old v0.Config
		if err := yaml.Unmarshal(data, &old); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
		config = v1.UpgradeFrom(old)

	default:
		config = latest.Config{}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig ensures the configuration is valid
func validateConfig(cfg *latest.Config) error {
	for _, model := range cfg.Models {
		if model.ParallelToolCalls == nil {
			model.ParallelToolCalls = boolPtr(true)
		}
	}

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

func boolPtr(b bool) *bool {
	return &b
}
