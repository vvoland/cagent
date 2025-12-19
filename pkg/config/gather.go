package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/gateway"
	"github.com/docker/cagent/pkg/model/provider"
)

// gatherMissingEnvVars finds out which environment variables are required by the models and tools.
// It returns the missing variables and any non-fatal error encountered during tool discovery.
func gatherMissingEnvVars(ctx context.Context, cfg *latest.Config, modelsGateway string, env environment.Provider) (missing []string, toolErr error) {
	requiredEnv := map[string]bool{}

	// Models
	if modelsGateway == "" {
		names := GatherEnvVarsForModels(cfg)
		for _, e := range names {
			requiredEnv[e] = true
		}
	}

	// Tools
	names, err := GatherEnvVarsForTools(ctx, cfg)
	if err != nil {
		// Store tool preflight error but continue checking models
		toolErr = err
	} else {
		for _, e := range names {
			requiredEnv[e] = true
		}
	}

	for _, e := range mcpToSortedList(requiredEnv) {
		if v, _ := env.Get(ctx, e); v == "" {
			missing = append(missing, e)
		}
	}

	return missing, toolErr
}

func GatherEnvVarsForModels(cfg *latest.Config) []string {
	requiredEnv := map[string]bool{}

	// Inspect only the models that are actually used by agents
	for agentName := range cfg.Agents {
		modelNames := strings.SplitSeq(cfg.Agents[agentName].Model, ",")
		for modelName := range modelNames {
			modelName = strings.TrimSpace(modelName)
			model := cfg.Models[modelName]

			if model.TokenKey != "" {
				requiredEnv[model.TokenKey] = true
			} else if alias, exists := provider.Aliases[model.Provider]; exists {
				// Use the token environment variable from the alias if available
				if alias.TokenEnvVar != "" {
					requiredEnv[alias.TokenEnvVar] = true
				}
			} else {
				// Fallback to hardcoded mappings for unknown providers
				switch model.Provider {
				case "openai":
					requiredEnv["OPENAI_API_KEY"] = true
				case "anthropic":
					requiredEnv["ANTHROPIC_API_KEY"] = true
				case "google":
					if model.ProviderOpts["project"] == nil && model.ProviderOpts["location"] == nil {
						requiredEnv["GOOGLE_API_KEY"] = true
					}
				case "mistral":
					requiredEnv["MISTRAL_API_KEY"] = true
				}
			}
		}
	}

	return mcpToSortedList(requiredEnv)
}

func GatherEnvVarsForTools(ctx context.Context, cfg *latest.Config) ([]string, error) {
	requiredEnv := map[string]bool{}
	var errs []error

	for i := range cfg.Agents {
		agent := cfg.Agents[i]

		for j := range agent.Toolsets {
			toolSet := agent.Toolsets[j]
			ref := toolSet.Ref
			if toolSet.Type != "mcp" || ref == "" {
				continue
			}

			mcpServerName := gateway.ParseServerRef(ref)
			secrets, err := gateway.RequiredEnvVars(ctx, mcpServerName)
			if err != nil {
				errs = append(errs, fmt.Errorf("reading which secrets the MCP server needs for %s: %w", ref, err))
				continue
			}

			for _, secret := range secrets {
				value, ok := toolSet.Env[secret.Env]
				if !ok {
					requiredEnv[secret.Env] = true
				} else {
					os.Expand(value, func(name string) string {
						requiredEnv[name] = true
						return ""
					})
				}
			}
		}
	}

	if len(errs) > 0 {
		return mcpToSortedList(requiredEnv), fmt.Errorf("tool env preflight: %w", errors.Join(errs...))
	}
	return mcpToSortedList(requiredEnv), nil
}

func mcpToSortedList(requiredEnv map[string]bool) []string {
	var requiredEnvList []string

	for e := range requiredEnv {
		requiredEnvList = append(requiredEnvList, e)
	}
	sort.Strings(requiredEnvList)

	return requiredEnvList
}
