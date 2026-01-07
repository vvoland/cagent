package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/modelsdev"
)

func TestResolveModelAliases(t *testing.T) {
	t.Parallel()

	// Create a mock database with sample models
	mockData := &modelsdev.Database{
		Providers: map[string]modelsdev.Provider{
			"anthropic": {
				ID:   "anthropic",
				Name: "Anthropic",
				Models: map[string]modelsdev.Model{
					"claude-sonnet-4-5":          {Name: "Claude Sonnet 4.5 (latest)"},
					"claude-sonnet-4-5-20250929": {Name: "Claude Sonnet 4.5"},
				},
			},
		},
	}

	store, err := modelsdev.NewStore(modelsdev.WithCacheDir(t.TempDir()))
	require.NoError(t, err)
	store.SetDatabaseForTesting(mockData)

	ctx := t.Context()

	tests := []struct {
		name     string
		cfg      *latest.Config
		expected *latest.Config
	}{
		{
			name: "resolves model in models section",
			cfg: &latest.Config{
				Models: map[string]latest.ModelConfig{
					"my_model": {Provider: "anthropic", Model: "claude-sonnet-4-5"},
				},
			},
			expected: &latest.Config{
				Models: map[string]latest.ModelConfig{
					"my_model": {Provider: "anthropic", Model: "claude-sonnet-4-5-20250929"},
				},
			},
		},
		{
			name: "resolves inline model in agent",
			cfg: &latest.Config{
				Models: map[string]latest.ModelConfig{},
				Agents: map[string]latest.AgentConfig{
					"root": {Model: "anthropic/claude-sonnet-4-5"},
				},
			},
			expected: &latest.Config{
				Models: map[string]latest.ModelConfig{},
				Agents: map[string]latest.AgentConfig{
					"root": {Model: "anthropic/claude-sonnet-4-5-20250929"},
				},
			},
		},
		{
			name: "resolves model config but not agent reference",
			cfg: &latest.Config{
				Models: map[string]latest.ModelConfig{
					"my_model": {Provider: "anthropic", Model: "claude-sonnet-4-5"},
				},
				Agents: map[string]latest.AgentConfig{
					"root": {Model: "my_model"},
				},
			},
			expected: &latest.Config{
				Models: map[string]latest.ModelConfig{
					"my_model": {Provider: "anthropic", Model: "claude-sonnet-4-5-20250929"},
				},
				Agents: map[string]latest.AgentConfig{
					"root": {Model: "my_model"},
				},
			},
		},
		{
			name: "keeps already pinned model unchanged",
			cfg: &latest.Config{
				Models: map[string]latest.ModelConfig{
					"my_model": {Provider: "anthropic", Model: "claude-sonnet-4-5-20250929"},
				},
			},
			expected: &latest.Config{
				Models: map[string]latest.ModelConfig{
					"my_model": {Provider: "anthropic", Model: "claude-sonnet-4-5-20250929"},
				},
			},
		},
		{
			name: "skips auto model",
			cfg: &latest.Config{
				Models: map[string]latest.ModelConfig{},
				Agents: map[string]latest.AgentConfig{
					"root": {Model: "auto"},
				},
			},
			expected: &latest.Config{
				Models: map[string]latest.ModelConfig{},
				Agents: map[string]latest.AgentConfig{
					"root": {Model: "auto"},
				},
			},
		},
		{
			name: "handles comma-separated models",
			cfg: &latest.Config{
				Models: map[string]latest.ModelConfig{},
				Agents: map[string]latest.AgentConfig{
					"root": {Model: "anthropic/claude-sonnet-4-5,my_ref"},
				},
			},
			expected: &latest.Config{
				Models: map[string]latest.ModelConfig{},
				Agents: map[string]latest.AgentConfig{
					"root": {Model: "anthropic/claude-sonnet-4-5-20250929,my_ref"},
				},
			},
		},
		{
			name: "resolves routing rules",
			cfg: &latest.Config{
				Models: map[string]latest.ModelConfig{
					"router": {
						Provider: "anthropic",
						Model:    "claude-sonnet-4-5",
						Routing: []latest.RoutingRule{
							{Model: "anthropic/claude-sonnet-4-5", Examples: []string{"example"}},
						},
					},
				},
			},
			expected: &latest.Config{
				Models: map[string]latest.ModelConfig{
					"router": {
						Provider: "anthropic",
						Model:    "claude-sonnet-4-5-20250929",
						Routing: []latest.RoutingRule{
							{Model: "anthropic/claude-sonnet-4-5-20250929", Examples: []string{"example"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResolveModelAliases(ctx, tt.cfg)
			assert.Equal(t, tt.expected, tt.cfg)
		})
	}
}
