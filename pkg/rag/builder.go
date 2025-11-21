package rag

import (
	"context"
	"fmt"
	"log/slog"

	latest "github.com/docker/cagent/pkg/config/v3"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/rag/strategy"
)

// ManagersBuildConfig contains dependencies needed to build RAG managers from config.
type ManagersBuildConfig struct {
	ParentDir     string
	ModelsGateway string
	Env           environment.Provider
}

// NewManagers constructs all RAG managers defined in the config.
func NewManagers(
	ctx context.Context,
	cfg *latest.Config,
	buildCfg ManagersBuildConfig,
) (map[string]*Manager, error) {
	managers := make(map[string]*Manager)

	if len(cfg.RAG) == 0 {
		return managers, nil
	}

	for ragName, ragCfg := range cfg.RAG {
		// Validate that we have at least one strategy
		if len(ragCfg.Strategies) == 0 {
			return nil, fmt.Errorf("no strategies configured for RAG %q", ragName)
		}

		// Build context for strategy builders
		strategyBuildCtx := strategy.BuildContext{
			RAGName:       ragName,
			ParentDir:     buildCfg.ParentDir,
			SharedDocs:    GetAbsolutePaths(buildCfg.ParentDir, ragCfg.Docs),
			Models:        cfg.Models,
			Env:           buildCfg.Env,
			ModelsGateway: buildCfg.ModelsGateway,
		}

		strategyConfigs, strategyEvents, err := buildStrategyConfigs(ctx, ragCfg, strategyBuildCtx, ragName)
		if err != nil {
			return nil, fmt.Errorf("failed to build strategy configs for RAG %q: %w", ragName, err)
		}

		managerCfg := buildManagerConfig(ragCfg, buildCfg.ParentDir, strategyConfigs)

		// The strategyEvents channel is so the manager can convert strategy events to RAG events.
		manager, err := New(ctx, ragName, managerCfg, strategyEvents)
		if err != nil {
			return nil, fmt.Errorf("failed to create RAG manager %q: %w", ragName, err)
		}

		managers[ragName] = manager

		strategyNames := make([]string, len(strategyConfigs))
		for i, sc := range strategyConfigs {
			strategyNames[i] = sc.Name
		}
		slog.Debug("Created RAG manager",
			"name", ragName,
			"strategies", strategyNames,
			"docs", len(managerCfg.Docs))
	}

	return managers, nil
}

// buildManagerConfig constructs a rag.Manager Config from the configuration and strategies.
func buildManagerConfig(
	ragCfg latest.RAGConfig,
	parentDir string,
	strategyConfigs []strategy.Config,
) Config {
	results := ResultsConfig{
		Limit:             ragCfg.Results.Limit,
		Deduplicate:       ragCfg.Results.Deduplicate,
		IncludeScore:      ragCfg.Results.IncludeScore,
		ReturnFullContent: ragCfg.Results.ReturnFullContent,
	}

	fusionCfg := buildManagerFusionConfig(ragCfg, strategyConfigs)

	return Config{
		Description:     ragCfg.Description,
		Docs:            GetAbsolutePaths(parentDir, ragCfg.Docs),
		Results:         results,
		FusionConfig:    fusionCfg,
		StrategyConfigs: strategyConfigs,
	}
}

// buildStrategyConfigs builds the strategy configs for the RAG.
// Returns a slice of strategy configs and a channel for receiving strategy events.
func buildStrategyConfigs(
	ctx context.Context,
	ragCfg latest.RAGConfig,
	strategyBuildCtx strategy.BuildContext,
	ragName string,
) ([]strategy.Config, chan strategy.Event, error) {
	// Create event channel for strategies to emit events.
	// This channel is shared with the manager, which exposes it directly to callers.
	// Use generous buffer to prevent blocking during heavy indexing.
	strategyEvents := make(chan strategy.Event, 500)

	// Build all strategies for this RAG source
	var strategyConfigs []strategy.Config
	for _, strategyCfg := range ragCfg.Strategies {
		builtStrategy, err := strategy.BuildStrategy(ctx, strategyCfg, strategyBuildCtx, strategyEvents)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build strategy %q for RAG %q: %w",
				strategyCfg.Type, ragName, err)
		}

		// Copy the built strategy config so callers can't mutate internal state.
		strategyConfigs = append(strategyConfigs, *builtStrategy)
	}

	return strategyConfigs, strategyEvents, nil
}

// buildManagerFusionConfig constructs a FusionConfig from the configuration.
// Returns nil if only one strategy is configured (no fusion needed).
func buildManagerFusionConfig(
	ragCfg latest.RAGConfig,
	strategyConfigs []strategy.Config,
) *FusionConfig {
	// Only use fusion if multiple strategies
	if len(strategyConfigs) <= 1 {
		return nil
	}

	fusionStrategy := "rrf" // Default to Reciprocal Rank Fusion
	fusionK := 60
	var fusionWeights map[string]float64

	if ragCfg.Results.Fusion != nil {
		if ragCfg.Results.Fusion.Strategy != "" {
			fusionStrategy = ragCfg.Results.Fusion.Strategy
		}
		if ragCfg.Results.Fusion.K > 0 {
			fusionK = ragCfg.Results.Fusion.K
		}
		fusionWeights = ragCfg.Results.Fusion.Weights
	}

	return &FusionConfig{
		Strategy: fusionStrategy,
		K:        fusionK,
		Weights:  fusionWeights,
	}
}
