package config

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/goccy/go-yaml"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/environment"
)

type Reader interface {
	Read(ctx context.Context) ([]byte, error)
}

func Load(ctx context.Context, source Reader) (*latest.Config, error) {
	data, err := source.Read(ctx)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Version string `yaml:"version,omitempty"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("looking for version in config file\n%s", yaml.FormatError(err, true, true))
	}
	if raw.Version == "" {
		raw.Version = latest.Version
	}

	oldConfig, err := parseCurrentVersion(data, raw.Version)
	if err != nil {
		return nil, fmt.Errorf("parsing config file\n%s", yaml.FormatError(err, true, true))
	}

	config, err := migrateToLatestConfig(oldConfig)
	if err != nil {
		return nil, fmt.Errorf("migrating config: %w", err)
	}

	config.Version = raw.Version

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// CheckRequiredEnvVars checks which environment variables are required by the models and tools.
//
// This allows exiting early with a proper error message instead of failing later when trying to use a model or tool.
func CheckRequiredEnvVars(ctx context.Context, cfg *latest.Config, modelsGateway string, env environment.Provider) error {
	missing, err := gatherMissingEnvVars(ctx, cfg, modelsGateway, env)
	if err != nil {
		// If there's a tool preflight error, log it but continue
		slog.Warn("Failed to preflight toolset environment variables; continuing", "error", err)
	}

	// Return error if there are missing environment variables
	if len(missing) > 0 {
		return &environment.RequiredEnvError{
			Missing: missing,
		}
	}

	return nil
}

func parseCurrentVersion(data []byte, version string) (any, error) {
	parser, found := Parsers()[version]
	if !found {
		return nil, fmt.Errorf("unsupported config version: %v", version)
	}
	return parser(data)
}

func migrateToLatestConfig(c any) (latest.Config, error) {
	var err error

	for _, upgrade := range Upgrades() {
		c, err = upgrade(c)
		if err != nil {
			return latest.Config{}, err
		}
	}

	return c.(latest.Config), nil
}

func validateConfig(cfg *latest.Config) error {
	if cfg.Models == nil {
		cfg.Models = map[string]latest.ModelConfig{}
	}

	for name := range cfg.Models {
		if cfg.Models[name].ParallelToolCalls == nil {
			m := cfg.Models[name]
			m.ParallelToolCalls = boolPtr(true)
			cfg.Models[name] = m
		}
	}

	if err := ensureModelsExist(cfg); err != nil {
		return err
	}

	for agentName := range cfg.Agents {
		agent := cfg.Agents[agentName]

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
