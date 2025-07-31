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

	// Migrate the configuration to the most recent version.
	oldConfig, err := parseCurrentVersion(data, raw["version"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	config := migrateToLatestConfig(oldConfig)

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func parseCurrentVersion(data []byte, version any) (any, error) {
	switch version {
	case nil, "0", 0:
		var old v0.Config
		if err := yaml.Unmarshal(data, &old); err != nil {
			return nil, err
		}
		return old, nil

	default:
		var old v1.Config
		if err := yaml.Unmarshal(data, &old); err != nil {
			return nil, err
		}
		return old, nil
	}
}

func migrateToLatestConfig(c any) latest.Config {
	for {
		if old, ok := c.(v0.Config); ok {
			c = v1.UpgradeFrom(old)
			continue
		}

		return c.(latest.Config)
	}
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
				// If the model is not found and is in the "provider/model" format, we can auto register.
				if provider, model, ok := strings.Cut(modelName, "/"); ok {
					autoRegisterModel(cfg, provider, model)
					continue
				}

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

// autoRegisterModel registers a model in the configuration if it does not exist.
func autoRegisterModel(cfg *latest.Config, provider, model string) {
	if cfg.Models == nil {
		cfg.Models = make(map[string]latest.ModelConfig)
	}

	cfg.Models[provider+"/"+model] = latest.ModelConfig{
		Provider: provider,
		Model:    model,
	}
}

func boolPtr(b bool) *bool {
	return &b
}
