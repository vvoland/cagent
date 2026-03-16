package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker-agent/pkg/config/latest"
	"github.com/docker/docker-agent/pkg/model/provider/options"
)

func TestApplyOverrides(t *testing.T) {
	t.Parallel()

	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name            string
		config          *latest.ModelConfig
		thinking        *bool // nil = no override
		wantBudget      *latest.ThinkingBudget
		wantInterleaved *bool // nil = key must not exist
	}{
		// --- Disable clears everything ---
		{
			name:     "disable: clears thinking_budget",
			config:   &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0", ThinkingBudget: &latest.ThinkingBudget{Tokens: 8192}},
			thinking: boolPtr(false),
		},
		{
			name:     "disable: clears interleaved_thinking",
			config:   &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0", ThinkingBudget: &latest.ThinkingBudget{Tokens: 16384}, ProviderOpts: map[string]any{"interleaved_thinking": true}},
			thinking: boolPtr(false),
		},

		// --- Enable preserves existing budget ---
		{
			name:       "enable: preserves existing budget",
			config:     &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0", ThinkingBudget: &latest.ThinkingBudget{Tokens: 8192}},
			thinking:   boolPtr(true),
			wantBudget: &latest.ThinkingBudget{Tokens: 8192},
		},
		{
			name:            "enable: preserves existing budget + interleaved",
			config:          &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0", ThinkingBudget: &latest.ThinkingBudget{Tokens: 8192}, ProviderOpts: map[string]any{"interleaved_thinking": true}},
			thinking:        boolPtr(true),
			wantBudget:      &latest.ThinkingBudget{Tokens: 8192},
			wantInterleaved: boolPtr(true),
		},

		// --- Enable applies defaults when no budget ---
		{
			name:       "enable: OpenAI gets medium default",
			config:     &latest.ModelConfig{Provider: "openai", Model: "gpt-4o"},
			thinking:   boolPtr(true),
			wantBudget: &latest.ThinkingBudget{Effort: "medium"},
		},
		{
			name:            "enable: Anthropic gets 8192 + interleaved",
			config:          &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0"},
			thinking:        boolPtr(true),
			wantBudget:      &latest.ThinkingBudget{Tokens: 8192},
			wantInterleaved: boolPtr(true),
		},
		{
			name:       "enable: restores from tokens=0",
			config:     &latest.ModelConfig{Provider: "openai", Model: "gpt-4o", ThinkingBudget: &latest.ThinkingBudget{Tokens: 0}},
			thinking:   boolPtr(true),
			wantBudget: &latest.ThinkingBudget{Effort: "medium"},
		},
		{
			name:            "enable: restores from effort=none",
			config:          &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0", ThinkingBudget: &latest.ThinkingBudget{Effort: "none"}},
			thinking:        boolPtr(true),
			wantBudget:      &latest.ThinkingBudget{Tokens: 8192},
			wantInterleaved: boolPtr(true),
		},

		// --- No override = no-op ---
		{
			name:       "nil opts: config unchanged",
			config:     &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0", ThinkingBudget: &latest.ThinkingBudget{Tokens: 8192}},
			thinking:   nil,
			wantBudget: &latest.ThinkingBudget{Tokens: 8192},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts *options.ModelOptions
			if tt.thinking != nil {
				o := options.ModelOptions{}
				options.WithThinking(*tt.thinking)(&o)
				opts = &o
			}

			result := applyOverrides(tt.config, opts)

			// Budget
			if tt.wantBudget == nil {
				assert.Nil(t, result.ThinkingBudget)
			} else {
				require.NotNil(t, result.ThinkingBudget)
				assert.Equal(t, *tt.wantBudget, *result.ThinkingBudget)
			}

			// interleaved_thinking
			if tt.wantInterleaved == nil && tt.thinking != nil && !*tt.thinking {
				if result.ProviderOpts != nil {
					_, exists := result.ProviderOpts["interleaved_thinking"]
					assert.False(t, exists, "interleaved_thinking should be removed")
				}
			} else if tt.wantInterleaved != nil {
				require.NotNil(t, result.ProviderOpts)
				assert.Equal(t, *tt.wantInterleaved, result.ProviderOpts["interleaved_thinking"])
			}
		})
	}
}

// TestApplyOverrides_DoesNotModifyOriginal verifies that applyOverrides creates
// a proper copy: neither the struct fields, the ProviderOpts map, nor the
// ThinkingBudget pointer of the original config are mutated.
func TestApplyOverrides_DoesNotModifyOriginal(t *testing.T) {
	t.Parallel()

	original := &latest.ModelConfig{
		Provider:       "anthropic",
		Model:          "claude-sonnet-4-0",
		ThinkingBudget: &latest.ThinkingBudget{Tokens: 8192},
		ProviderOpts:   map[string]any{"interleaved_thinking": true, "other": "value"},
	}

	o := options.ModelOptions{}
	options.WithThinking(false)(&o)
	result := applyOverrides(original, &o)

	// Result should have thinking cleared.
	assert.Nil(t, result.ThinkingBudget, "result ThinkingBudget should be nil")

	// Original ThinkingBudget must be untouched.
	require.NotNil(t, original.ThinkingBudget, "original ThinkingBudget must survive")
	assert.Equal(t, 8192, original.ThinkingBudget.Tokens)

	// Original ProviderOpts map must still have interleaved_thinking.
	val, exists := original.ProviderOpts["interleaved_thinking"]
	require.True(t, exists, "original ProviderOpts must still contain interleaved_thinking")
	assert.Equal(t, true, val)

	// Other keys must survive in both original and result.
	assert.Equal(t, "value", original.ProviderOpts["other"])
	require.NotNil(t, result.ProviderOpts)
	assert.Equal(t, "value", result.ProviderOpts["other"])
}

// TestApplyOverrides_DisablePreservesOtherProviderOpts verifies that disabling
// thinking only removes "interleaved_thinking" and leaves other keys intact.
func TestApplyOverrides_DisablePreservesOtherProviderOpts(t *testing.T) {
	t.Parallel()

	config := &latest.ModelConfig{
		Provider:       "anthropic",
		Model:          "claude-sonnet-4-0",
		ThinkingBudget: &latest.ThinkingBudget{Tokens: 8192},
		ProviderOpts:   map[string]any{"interleaved_thinking": true, "custom_key": "preserved"},
	}

	o := options.ModelOptions{}
	options.WithThinking(false)(&o)
	result := applyOverrides(config, &o)

	// Thinking should be cleared.
	assert.Nil(t, result.ThinkingBudget)

	// interleaved_thinking should be removed.
	_, exists := result.ProviderOpts["interleaved_thinking"]
	assert.False(t, exists, "interleaved_thinking should be removed from result")

	// Other keys must survive.
	assert.Equal(t, "preserved", result.ProviderOpts["custom_key"])
}

// TestDefaultsThenOverrides tests the complete flow: provider defaults → overrides.
func TestDefaultsThenOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     *latest.ModelConfig
		thinking   bool
		wantBudget *latest.ThinkingBudget
	}{
		// Disable on models without defaults — already nil, stays nil.
		{"gpt-4o /think off", &latest.ModelConfig{Provider: "openai", Model: "gpt-4o"}, false, nil},
		{"anthropic /think off", &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0"}, false, nil},

		// Enable on models without defaults — applies provider defaults.
		{"gpt-4o /think on", &latest.ModelConfig{Provider: "openai", Model: "gpt-4o"}, true, &latest.ThinkingBudget{Effort: "medium"}},
		{"anthropic /think on", &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0"}, true, &latest.ThinkingBudget{Tokens: 8192}},
		{"gemini-2.5 /think on", &latest.ModelConfig{Provider: "google", Model: "gemini-2.5-flash"}, true, &latest.ThinkingBudget{Tokens: -1}},
		{"gemini-3-pro /think on", &latest.ModelConfig{Provider: "google", Model: "gemini-3-pro"}, true, &latest.ThinkingBudget{Effort: "high"}},
		{"gemini-3-flash /think on", &latest.ModelConfig{Provider: "google", Model: "gemini-3-flash"}, true, &latest.ThinkingBudget{Effort: "medium"}},
		{"bedrock claude /think on", &latest.ModelConfig{Provider: "amazon-bedrock", Model: "anthropic.claude-3-sonnet"}, true, &latest.ThinkingBudget{Tokens: 8192}},

		// Old Gemini model that doesn't support thinking — /think should be a no-op.
		{"gemini-2.0 /think on (no thinking support)", &latest.ModelConfig{Provider: "google", Model: "gemini-2.0-flash"}, true, nil},

		// Thinking-only model defaults preserved when enabled, cleared when disabled.
		{"o3-mini /think on", &latest.ModelConfig{Provider: "openai", Model: "o3-mini"}, true, &latest.ThinkingBudget{Effort: "medium"}},
		{"o3-mini /think off", &latest.ModelConfig{Provider: "openai", Model: "o3-mini"}, false, nil},

		// Explicit budget cleared by disable.
		{"explicit cleared", &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0", ThinkingBudget: &latest.ThinkingBudget{Tokens: 32000}}, false, nil},

		// Restore from disabled (thinking_budget: 0) via /think on.
		{"restore from 0", &latest.ModelConfig{Provider: "openai", Model: "gpt-4o", ThinkingBudget: &latest.ThinkingBudget{Tokens: 0}}, true, &latest.ThinkingBudget{Effort: "medium"}},
		{"restore from none", &latest.ModelConfig{Provider: "anthropic", Model: "claude-sonnet-4-0", ThinkingBudget: &latest.ThinkingBudget{Effort: "none"}}, true, &latest.ThinkingBudget{Tokens: 8192}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := applyProviderDefaults(tt.config, nil)

			o := options.ModelOptions{}
			options.WithThinking(tt.thinking)(&o)
			result = applyOverrides(result, &o)

			if tt.wantBudget == nil {
				assert.Nil(t, result.ThinkingBudget)
			} else {
				require.NotNil(t, result.ThinkingBudget)
				assert.Equal(t, *tt.wantBudget, *result.ThinkingBudget)
			}
		})
	}
}

// TestApplyProviderDefaults_DoesNotModifyOriginal verifies that applyProviderDefaults
// does not mutate the input config's ProviderOpts map.
func TestApplyProviderDefaults_DoesNotModifyOriginal(t *testing.T) {
	t.Parallel()

	original := &latest.ModelConfig{
		Provider:       "anthropic",
		Model:          "claude-sonnet-4-0",
		ThinkingBudget: &latest.ThinkingBudget{Tokens: 8192},
		ProviderOpts:   map[string]any{"custom_key": "original_value"},
	}

	result := applyProviderDefaults(original, nil)

	// Result should have interleaved_thinking set (because thinking_budget is set).
	require.NotNil(t, result.ProviderOpts)
	assert.Equal(t, true, result.ProviderOpts["interleaved_thinking"])

	// Original must NOT have interleaved_thinking added.
	_, exists := original.ProviderOpts["interleaved_thinking"]
	assert.False(t, exists, "original ProviderOpts must not be mutated by applyProviderDefaults")

	// Original custom key must still be there.
	assert.Equal(t, "original_value", original.ProviderOpts["custom_key"])
}
