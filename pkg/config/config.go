package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v0 "github.com/docker/cagent/pkg/config/v0"
	v1 "github.com/docker/cagent/pkg/config/v1"
	"gopkg.in/yaml.v3"
)

// LoadConfigSecure loads the configuration from a file with path validation
func LoadConfigSecure(path string, allowedDir string) (*v1.Config, error) {
	validatedPath, err := validatePathInDirectory(path, allowedDir)
	if err != nil {
		return nil, fmt.Errorf("path validation failed: %w", err)
	}

	return LoadConfig(validatedPath)
}

func validatePathInDirectory(path, allowedDir string) (string, error) {
	cleanAllowedDir := filepath.Clean(allowedDir)
	absAllowedDir, err := filepath.Abs(cleanAllowedDir)
	if err != nil {
		return "", fmt.Errorf("invalid allowed directory: %w", err)
	}

	var targetPath string
	if filepath.IsAbs(path) {
		targetPath = filepath.Clean(path)
	} else {
		targetPath = filepath.Join(absAllowedDir, path)
	}

	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	relPath, err := filepath.Rel(absAllowedDir, absTargetPath)
	if err != nil {
		return "", fmt.Errorf("cannot determine relative path: %w", err)
	}

	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path outside allowed directory: %s", path)
	}

	return absTargetPath, nil
}

func LoadConfig(path string) (*v1.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

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

func migrateToLatestConfig(c any) v1.Config {
	for {
		if old, ok := c.(v0.Config); ok {
			c = v1.UpgradeFrom(old)
			continue
		}

		return c.(v1.Config)
	}
}

func validateConfig(cfg *v1.Config) error {
	for _, model := range cfg.Models {
		if model.ParallelToolCalls == nil {
			model.ParallelToolCalls = boolPtr(true)
		}
	}

	for agentName := range cfg.Agents {
		agent := cfg.Agents[agentName]

		modelNames := strings.SplitSeq(agent.Model, ",")
		for modelName := range modelNames {
			if _, exists := cfg.Models[modelName]; !exists {
				if provider, model, ok := strings.Cut(modelName, "/"); ok {
					autoRegisterModel(cfg, provider, model)
					continue
				}

				return fmt.Errorf("agent '%s' references non-existent model '%s'", agentName, modelName)
			}
		}

		for _, subAgentName := range agent.SubAgents {
			if _, exists := cfg.Agents[subAgentName]; !exists {
				return fmt.Errorf("agent '%s' references non-existent sub-agent '%s'", agentName, subAgentName)
			}
		}
	}

	return nil
}

func autoRegisterModel(cfg *v1.Config, provider, model string) {
	if cfg.Models == nil {
		cfg.Models = make(map[string]v1.ModelConfig)
	}

	cfg.Models[provider+"/"+model] = v1.ModelConfig{
		Provider: provider,
		Model:    model,
	}
}

func boolPtr(b bool) *bool {
	return &b
}
