package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker-agent/pkg/config/latest"
	"github.com/docker/docker-agent/pkg/model/provider/base"
)

func TestAnthropicThinkingDisplay(t *testing.T) {
	tests := []struct {
		name   string
		opts   map[string]any
		want   string
		wantOk bool
	}{
		{
			name:   "nil opts",
			opts:   nil,
			want:   "",
			wantOk: false,
		},
		{
			name:   "empty opts",
			opts:   map[string]any{},
			want:   "",
			wantOk: false,
		},
		{
			name:   "key missing",
			opts:   map[string]any{"other": "foo"},
			want:   "",
			wantOk: false,
		},
		{
			name:   "summarized",
			opts:   map[string]any{"thinking_display": "summarized"},
			want:   "summarized",
			wantOk: true,
		},
		{
			name:   "omitted",
			opts:   map[string]any{"thinking_display": "omitted"},
			want:   "omitted",
			wantOk: true,
		},
		{
			name:   "display",
			opts:   map[string]any{"thinking_display": "display"},
			want:   "display",
			wantOk: true,
		},
		{
			name:   "case insensitive",
			opts:   map[string]any{"thinking_display": "SUMMARIZED"},
			want:   "summarized",
			wantOk: true,
		},
		{
			name:   "whitespace trimmed",
			opts:   map[string]any{"thinking_display": "  omitted  "},
			want:   "omitted",
			wantOk: true,
		},
		{
			name:   "invalid string",
			opts:   map[string]any{"thinking_display": "not-a-valid-value"},
			want:   "",
			wantOk: false,
		},
		{
			name:   "non-string value",
			opts:   map[string]any{"thinking_display": 42},
			want:   "",
			wantOk: false,
		},
		{
			name:   "bool value",
			opts:   map[string]any{"thinking_display": true},
			want:   "",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := anthropicThinkingDisplay(tt.opts)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// clientWith builds a minimal Client with the given ThinkingBudget and
// provider_opts for use in thinking-config tests.
func clientWith(budget *latest.ThinkingBudget, opts map[string]any) *Client {
	return &Client{
		Config: base.Config{
			ModelConfig: latest.ModelConfig{
				Provider:       "anthropic",
				Model:          "claude-opus-4-7",
				ThinkingBudget: budget,
				ProviderOpts:   opts,
			},
		},
	}
}

func TestApplyThinkingConfig(t *testing.T) {
	tests := []struct {
		name            string
		budget          *latest.ThinkingBudget
		opts            map[string]any
		maxTokens       int64
		wantEnabled     bool
		wantAdaptive    bool
		wantTokens      int64
		wantEffort      string
		wantDisplayJSON string // "" means the display field must not be present in JSON
	}{
		{
			name:        "nil budget disables thinking",
			budget:      nil,
			maxTokens:   8192,
			wantEnabled: false,
		},
		{
			name:        "token budget below minimum is ignored",
			budget:      &latest.ThinkingBudget{Tokens: 500},
			maxTokens:   8192,
			wantEnabled: false,
		},
		{
			name:        "token budget above max_tokens is ignored",
			budget:      &latest.ThinkingBudget{Tokens: 9000},
			maxTokens:   8192,
			wantEnabled: false,
		},
		{
			name:         "adaptive high effort without display",
			budget:       &latest.ThinkingBudget{Effort: "adaptive"},
			maxTokens:    8192,
			wantEnabled:  true,
			wantAdaptive: true,
			wantEffort:   "high",
		},
		{
			name:            "adaptive with display=summarized",
			budget:          &latest.ThinkingBudget{Effort: "adaptive"},
			opts:            map[string]any{"thinking_display": "summarized"},
			maxTokens:       8192,
			wantEnabled:     true,
			wantAdaptive:    true,
			wantEffort:      "high",
			wantDisplayJSON: "summarized",
		},
		{
			name:            "adaptive with display=omitted",
			budget:          &latest.ThinkingBudget{Effort: "adaptive/low"},
			opts:            map[string]any{"thinking_display": "omitted"},
			maxTokens:       8192,
			wantEnabled:     true,
			wantAdaptive:    true,
			wantEffort:      "low",
			wantDisplayJSON: "omitted",
		},
		{
			name:        "token budget without display",
			budget:      &latest.ThinkingBudget{Tokens: 2048},
			maxTokens:   8192,
			wantEnabled: true,
			wantTokens:  2048,
		},
		{
			name:            "token budget with display=display",
			budget:          &latest.ThinkingBudget{Tokens: 2048},
			opts:            map[string]any{"thinking_display": "display"},
			maxTokens:       8192,
			wantEnabled:     true,
			wantTokens:      2048,
			wantDisplayJSON: "display",
		},
		{
			name:        "invalid display value is ignored",
			budget:      &latest.ThinkingBudget{Tokens: 2048},
			opts:        map[string]any{"thinking_display": "bogus"},
			maxTokens:   8192,
			wantEnabled: true,
			wantTokens:  2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clientWith(tt.budget, tt.opts)
			params := anthropic.MessageNewParams{}

			gotEnabled := c.applyThinkingConfig(&params, tt.maxTokens)
			assert.Equal(t, tt.wantEnabled, gotEnabled)

			if !tt.wantEnabled {
				assert.Nil(t, params.Thinking.OfAdaptive)
				assert.Nil(t, params.Thinking.OfEnabled)
				return
			}

			if tt.wantAdaptive {
				require.NotNil(t, params.Thinking.OfAdaptive)
				assert.Equal(t, tt.wantEffort, string(params.OutputConfig.Effort))
				assert.Equal(t, tt.wantDisplayJSON, string(params.Thinking.OfAdaptive.Display))
			} else {
				require.NotNil(t, params.Thinking.OfEnabled)
				assert.Equal(t, tt.wantTokens, params.Thinking.OfEnabled.BudgetTokens)
				assert.Equal(t, tt.wantDisplayJSON, string(params.Thinking.OfEnabled.Display))
			}

			// Sanity-check: the marshaled JSON omits display entirely when unset,
			// thanks to the SDK's `json:"display,omitzero"` tag.
			b, err := json.Marshal(params.Thinking)
			require.NoError(t, err)
			if tt.wantDisplayJSON == "" {
				assert.NotContains(t, string(b), `"display"`)
			} else {
				assert.Contains(t, string(b), `"display":"`+tt.wantDisplayJSON+`"`)
			}
		})
	}
}

func TestApplyBetaThinkingConfig(t *testing.T) {
	tests := []struct {
		name            string
		budget          *latest.ThinkingBudget
		opts            map[string]any
		maxTokens       int64
		wantAdaptive    bool
		wantEnabled     bool
		wantTokens      int64
		wantEffort      string
		wantDisplayJSON string
	}{
		{
			name:      "nil budget leaves params untouched",
			budget:    nil,
			maxTokens: 8192,
		},
		{
			name:            "adaptive with display",
			budget:          &latest.ThinkingBudget{Effort: "adaptive/medium"},
			opts:            map[string]any{"thinking_display": "display"},
			maxTokens:       8192,
			wantAdaptive:    true,
			wantEffort:      "medium",
			wantDisplayJSON: "display",
		},
		{
			name:            "token budget with display=omitted",
			budget:          &latest.ThinkingBudget{Tokens: 4096},
			opts:            map[string]any{"thinking_display": "omitted"},
			maxTokens:       8192,
			wantEnabled:     true,
			wantTokens:      4096,
			wantDisplayJSON: "omitted",
		},
		{
			name:      "invalid token budget leaves params untouched",
			budget:    &latest.ThinkingBudget{Tokens: 100},
			maxTokens: 8192,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clientWith(tt.budget, tt.opts)
			params := anthropic.BetaMessageNewParams{}

			c.applyBetaThinkingConfig(&params, tt.maxTokens)

			switch {
			case tt.wantAdaptive:
				require.NotNil(t, params.Thinking.OfAdaptive)
				assert.Equal(t, tt.wantEffort, string(params.OutputConfig.Effort))
				assert.Equal(t, tt.wantDisplayJSON, string(params.Thinking.OfAdaptive.Display))
			case tt.wantEnabled:
				require.NotNil(t, params.Thinking.OfEnabled)
				assert.Equal(t, tt.wantTokens, params.Thinking.OfEnabled.BudgetTokens)
				assert.Equal(t, tt.wantDisplayJSON, string(params.Thinking.OfEnabled.Display))
			default:
				assert.Nil(t, params.Thinking.OfAdaptive)
				assert.Nil(t, params.Thinking.OfEnabled)
			}
		})
	}
}

func TestAdjustMaxTokensForThinking(t *testing.T) {
	t.Run("no budget returns input unchanged", func(t *testing.T) {
		c := clientWith(nil, nil)
		got, err := c.adjustMaxTokensForThinking(8192)
		require.NoError(t, err)
		assert.Equal(t, int64(8192), got)
	})

	t.Run("adaptive budget returns input unchanged", func(t *testing.T) {
		c := clientWith(&latest.ThinkingBudget{Effort: "adaptive"}, nil)
		got, err := c.adjustMaxTokensForThinking(8192)
		require.NoError(t, err)
		assert.Equal(t, int64(8192), got)
	})

	t.Run("token budget fits in max_tokens", func(t *testing.T) {
		c := clientWith(&latest.ThinkingBudget{Tokens: 2048}, nil)
		got, err := c.adjustMaxTokensForThinking(8192)
		require.NoError(t, err)
		assert.Equal(t, int64(8192), got)
	})

	t.Run("auto-adjust when user didn't set max_tokens", func(t *testing.T) {
		c := clientWith(&latest.ThinkingBudget{Tokens: 16384}, nil)
		got, err := c.adjustMaxTokensForThinking(8192)
		require.NoError(t, err)
		assert.Equal(t, int64(16384+8192), got)
	})

	t.Run("error when user explicitly set max_tokens too low", func(t *testing.T) {
		c := clientWith(&latest.ThinkingBudget{Tokens: 16384}, nil)
		userMax := int64(8192)
		c.ModelConfig.MaxTokens = &userMax
		_, err := c.adjustMaxTokensForThinking(8192)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_tokens")
	})
}

func TestInterleavedThinkingEnabled(t *testing.T) {
	tests := []struct {
		name string
		opts map[string]any
		want bool
	}{
		{"nil opts", nil, false},
		{"missing key", map[string]any{"other": true}, false},
		{"bool true", map[string]any{"interleaved_thinking": true}, true},
		{"bool false", map[string]any{"interleaved_thinking": false}, false},
		{"string true", map[string]any{"interleaved_thinking": "true"}, true},
		{"string false", map[string]any{"interleaved_thinking": "false"}, false},
		{"string no", map[string]any{"interleaved_thinking": "no"}, false},
		{"int 0", map[string]any{"interleaved_thinking": 0}, false},
		{"int 1", map[string]any{"interleaved_thinking": 1}, true},
		{"unsupported type", map[string]any{"interleaved_thinking": []string{}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clientWith(nil, tt.opts)
			assert.Equal(t, tt.want, c.interleavedThinkingEnabled())
		})
	}
}
