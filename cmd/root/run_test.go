package root

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/teamloader"
)

func TestComputeInitialThinking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		loadResult *teamloader.LoadResult
		agentName  string
		expected   bool
	}{
		{
			name: "v4 config - thinking enabled by default",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "4",
				AgentDefaultModels: map[string]string{"root": "my_model"},
				Models:             map[string]latest.ModelConfig{"my_model": {Provider: "openai", Model: "gpt-4o"}},
			},
			agentName: "root",
			expected:  true,
		},
		{
			name: "empty version (latest) - thinking enabled by default",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "",
				AgentDefaultModels: map[string]string{"root": "my_model"},
				Models:             map[string]latest.ModelConfig{"my_model": {Provider: "openai", Model: "gpt-4o"}},
			},
			agentName: "root",
			expected:  true,
		},
		{
			name: "v3 config - no thinking_budget - disabled",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "3",
				AgentDefaultModels: map[string]string{"root": "my_model"},
				Models:             map[string]latest.ModelConfig{"my_model": {Provider: "openai", Model: "gpt-4o"}},
			},
			agentName: "root",
			expected:  false,
		},
		{
			name: "v3 config - thinking_budget explicitly set - enabled",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "3",
				AgentDefaultModels: map[string]string{"root": "my_model"},
				Models: map[string]latest.ModelConfig{
					"my_model": {
						Provider:       "openai",
						Model:          "gpt-4o",
						ThinkingBudget: &latest.ThinkingBudget{Effort: "medium"},
					},
				},
			},
			agentName: "root",
			expected:  true,
		},
		{
			name: "v3 config - thinking_budget explicitly disabled with tokens=0",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "3",
				AgentDefaultModels: map[string]string{"root": "my_model"},
				Models: map[string]latest.ModelConfig{
					"my_model": {
						Provider:       "anthropic",
						Model:          "claude-sonnet-4-0",
						ThinkingBudget: &latest.ThinkingBudget{Tokens: 0},
					},
				},
			},
			agentName: "root",
			expected:  false,
		},
		{
			name: "v3 config - thinking_budget explicitly disabled with effort=none",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "3",
				AgentDefaultModels: map[string]string{"root": "my_model"},
				Models: map[string]latest.ModelConfig{
					"my_model": {
						Provider:       "openai",
						Model:          "gpt-4o",
						ThinkingBudget: &latest.ThinkingBudget{Effort: "none"},
					},
				},
			},
			agentName: "root",
			expected:  false,
		},
		{
			name: "v0 config - inline model spec - disabled (no explicit config)",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "0",
				AgentDefaultModels: map[string]string{"root": "openai/gpt-4o"},
				Models:             map[string]latest.ModelConfig{}, // inline specs won't be in Models
			},
			agentName: "root",
			expected:  false,
		},
		{
			name: "v1 config - no model reference - disabled",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "1",
				AgentDefaultModels: map[string]string{}, // missing agent
				Models:             map[string]latest.ModelConfig{},
			},
			agentName: "root",
			expected:  false,
		},
		{
			name: "v2 config - comma-separated models - checks first model",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "2",
				AgentDefaultModels: map[string]string{"root": "model_a,model_b"},
				Models: map[string]latest.ModelConfig{
					"model_a": {
						Provider:       "openai",
						Model:          "gpt-4o",
						ThinkingBudget: &latest.ThinkingBudget{Effort: "high"},
					},
					"model_b": {
						Provider: "anthropic",
						Model:    "claude-sonnet-4-0",
					},
				},
			},
			agentName: "root",
			expected:  true, // first model has thinking enabled
		},
		{
			name: "v3 config - thinking_budget with tokens > 0 - enabled",
			loadResult: &teamloader.LoadResult{
				ConfigVersion:      "3",
				AgentDefaultModels: map[string]string{"root": "my_model"},
				Models: map[string]latest.ModelConfig{
					"my_model": {
						Provider:       "anthropic",
						Model:          "claude-sonnet-4-0",
						ThinkingBudget: &latest.ThinkingBudget{Tokens: 8192},
					},
				},
			},
			agentName: "root",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := computeInitialThinking(tt.loadResult, tt.agentName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestThinkingDisabledForOldConfigVersions specifically tests that thinking is disabled
// for v1, v2, v3 configs when thinking_budget is NOT defined, even with thinking-capable models.
// This is the key behavioral change: old configs default to thinking=off.
func TestThinkingDisabledForOldConfigVersions(t *testing.T) {
	t.Parallel()

	// All these models would normally get thinking defaults in v4,
	// but should have thinking DISABLED in v1/v2/v3 when not explicitly configured.
	thinkingCapableModels := []latest.ModelConfig{
		{Provider: "openai", Model: "gpt-4o"},
		{Provider: "openai", Model: "gpt-5"},
		{Provider: "anthropic", Model: "claude-sonnet-4-0"},
		{Provider: "google", Model: "gemini-2.5-pro"},
		{Provider: "amazon-bedrock", Model: "anthropic.claude-3-sonnet"},
	}

	oldVersions := []string{"0", "1", "2", "3"}

	for _, version := range oldVersions {
		for _, modelCfg := range thinkingCapableModels {
			modelName := modelCfg.Provider + "/" + modelCfg.Model
			t.Run("v"+version+"_"+modelName+"_no_thinking_budget", func(t *testing.T) {
				t.Parallel()

				loadResult := &teamloader.LoadResult{
					ConfigVersion:      version,
					AgentDefaultModels: map[string]string{"root": "my_model"},
					Models: map[string]latest.ModelConfig{
						"my_model": modelCfg, // No ThinkingBudget set
					},
				}

				result := computeInitialThinking(loadResult, "root")
				assert.False(t, result,
					"v%s config with %s should have thinking DISABLED when thinking_budget is not defined",
					version, modelName)
			})
		}
	}
}

// TestThinkingEnabledForV4 verifies that v4 configs get thinking enabled by default
// (the opposite of old configs).
func TestThinkingEnabledForV4(t *testing.T) {
	t.Parallel()

	loadResult := &teamloader.LoadResult{
		ConfigVersion:      "4",
		AgentDefaultModels: map[string]string{"root": "my_model"},
		Models: map[string]latest.ModelConfig{
			"my_model": {Provider: "openai", Model: "gpt-5"}, // No ThinkingBudget set
		},
	}

	result := computeInitialThinking(loadResult, "root")
	assert.True(t, result, "v4 config should have thinking ENABLED by default")
}

// TestThinkingExplicitlySetInOldConfigs verifies that when a user explicitly sets
// thinking_budget in an old config, it IS respected and thinking is enabled.
func TestThinkingExplicitlySetInOldConfigs(t *testing.T) {
	t.Parallel()

	oldVersions := []string{"1", "2", "3"}
	explicitConfigs := []latest.ThinkingBudget{
		{Effort: "medium"},
		{Effort: "high"},
		{Tokens: 8192},
		{Tokens: 16000},
	}

	for _, version := range oldVersions {
		for _, budget := range explicitConfigs {
			budgetDesc := ""
			if budget.Effort != "" {
				budgetDesc = "effort_" + budget.Effort
			} else {
				budgetDesc = "tokens_" + string(rune(budget.Tokens))
			}
			t.Run("v"+version+"_explicit_"+budgetDesc, func(t *testing.T) {
				t.Parallel()

				loadResult := &teamloader.LoadResult{
					ConfigVersion:      version,
					AgentDefaultModels: map[string]string{"root": "my_model"},
					Models: map[string]latest.ModelConfig{
						"my_model": {
							Provider:       "openai",
							Model:          "gpt-4o",
							ThinkingBudget: &budget,
						},
					},
				}

				result := computeInitialThinking(loadResult, "root")
				assert.True(t, result,
					"v%s config with explicit thinking_budget should have thinking ENABLED",
					version)
			})
		}
	}
}
