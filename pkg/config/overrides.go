package config

import (
	"fmt"
	"strings"

	v2 "github.com/docker/cagent/pkg/config/v2"
)

// ApplyModelOverrides applies CLI model overrides to the configuration
func ApplyModelOverrides(cfg *v2.Config, overrides []string) error {
	for _, override := range overrides {
		if err := applySingleOverride(cfg, override); err != nil {
			return err
		}
	}

	// After applying overrides, ensure new models are added to cfg.Models
	return ensureModelsExist(cfg)
}

// applySingleOverride processes a single model override string
func applySingleOverride(cfg *v2.Config, override string) error {
	override = strings.TrimSpace(override)
	if override == "" {
		return nil // Skip empty overrides
	}

	// Handle comma-separated format: "agent1=model1,agent2=model2"
	if strings.Contains(override, ",") {
		for part := range strings.SplitSeq(override, ",") {
			if err := applySingleOverride(cfg, part); err != nil {
				return err
			}
		}
		return nil
	}

	// Check if this is an agent-specific override (contains '=')
	agentName, modelSpec, ok := strings.Cut(override, "=")
	if ok {
		agentName = strings.TrimSpace(agentName)
		if agentName == "" {
			return fmt.Errorf("empty agent name in override: %s", override)
		}

		modelSpec = strings.TrimSpace(modelSpec)
		if modelSpec == "" {
			return fmt.Errorf("empty model specification in override: %s", override)
		}

		// Apply to specific agent
		agentConfig, exists := cfg.Agents[agentName]
		if !exists {
			return fmt.Errorf("unknown agent '%s'", agentName)
		}

		agentConfig.Model = modelSpec
		cfg.Agents[agentName] = agentConfig
	} else {
		// Global override: apply to all agents
		modelSpec := strings.TrimSpace(override)
		if modelSpec == "" {
			return fmt.Errorf("empty model specification")
		}

		for name := range cfg.Agents {
			agentConfig := cfg.Agents[name]
			agentConfig.Model = modelSpec
			cfg.Agents[name] = agentConfig
		}
	}

	return nil
}

// ensureModelsExist ensures that all models referenced by agents exist in cfg.Models
// This handles inline model specs that may have been added via CLI overrides
func ensureModelsExist(cfg *v2.Config) error {
	if cfg.Models == nil {
		cfg.Models = map[string]v2.ModelConfig{}
	}

	// Ensure models referenced by agents exist
	for agentName := range cfg.Agents {
		agentConfig := cfg.Agents[agentName]

		modelNames := strings.SplitSeq(agentConfig.Model, ",")
		for modelName := range modelNames {
			if err := ensureSingleModelExists(cfg, modelName, fmt.Sprintf("agent '%s'", agentName)); err != nil {
				return err
			}
		}
	}

	// Ensure models referenced by RAG strategies exist
	for ragName, ragCfg := range cfg.RAG {
		for _, stratCfg := range ragCfg.Strategies {
			rawModel, ok := stratCfg.Params["model"]
			if !ok {
				continue
			}

			modelName, ok := rawModel.(string)
			if !ok {
				return fmt.Errorf("RAG strategy '%s' in RAG '%s' has non-string model value", stratCfg.Type, ragName)
			}

			if err := ensureSingleModelExists(cfg, modelName, fmt.Sprintf("RAG strategy '%s' in RAG '%s'", stratCfg.Type, ragName)); err != nil {
				return err
			}
		}
	}

	return nil
}

// ensureSingleModelExists normalizes shorthand model IDs like "openai/gpt-5-mini"
// into full entries in cfg.Models so they can be reused by agents, RAG, and other
// subsystems without duplicating parsing logic.
func ensureSingleModelExists(cfg *v2.Config, modelName, context string) error {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || modelName == "auto" {
		// "auto" is handled dynamically at runtime and does not need a config entry.
		return nil
	}

	if _, exists := cfg.Models[modelName]; exists {
		return nil
	}

	providerName, model, ok := strings.Cut(modelName, "/")
	if !ok || providerName == "" || model == "" {
		return fmt.Errorf("%s references non-existent model '%s'", context, modelName)
	}

	cfg.Models[modelName] = v2.ModelConfig{
		Provider: providerName,
		Model:    model,
	}

	return nil
}
