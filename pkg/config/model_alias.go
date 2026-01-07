package config

import (
	"context"
	"log/slog"
	"strings"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/modelsdev"
)

// ResolveModelAliases resolves model aliases to their pinned versions in the config.
// For example, "claude-sonnet-4-5" might resolve to "claude-sonnet-4-5-20250929".
// This modifies the config in place.
func ResolveModelAliases(ctx context.Context, cfg *latest.Config) {
	store, err := modelsdev.NewStore()
	if err != nil {
		slog.Debug("Failed to create modelsdev store for alias resolution", "error", err)
		return
	}

	// Resolve model aliases in the models section
	for name, modelCfg := range cfg.Models {
		if resolved := store.ResolveModelAlias(ctx, modelCfg.Provider, modelCfg.Model); resolved != modelCfg.Model {
			modelCfg.Model = resolved
			cfg.Models[name] = modelCfg
		}

		// Resolve model aliases in routing rules
		for i, rule := range modelCfg.Routing {
			if provider, model, ok := strings.Cut(rule.Model, "/"); ok {
				if resolved := store.ResolveModelAlias(ctx, provider, model); resolved != model {
					modelCfg.Routing[i].Model = provider + "/" + resolved
				}
			}
		}
		cfg.Models[name] = modelCfg
	}

	// Resolve inline model references in agents (e.g., "anthropic/claude-sonnet-4-5")
	for agentName, agentCfg := range cfg.Agents {
		if agentCfg.Model == "" || agentCfg.Model == "auto" {
			continue
		}

		var resolvedModels []string
		for modelRef := range strings.SplitSeq(agentCfg.Model, ",") {
			if provider, model, ok := strings.Cut(modelRef, "/"); ok {
				if resolved := store.ResolveModelAlias(ctx, provider, model); resolved != model {
					resolvedModels = append(resolvedModels, provider+"/"+resolved)
					continue
				}
			}
			resolvedModels = append(resolvedModels, modelRef)
		}

		agentCfg.Model = strings.Join(resolvedModels, ",")
		cfg.Agents[agentName] = agentCfg
	}
}
