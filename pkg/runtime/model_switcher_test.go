package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInlineAlloySpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		modelRef string
		want     bool
	}{
		{
			name:     "single inline model",
			modelRef: "openai/gpt-4o",
			want:     false,
		},
		{
			name:     "two inline models",
			modelRef: "openai/gpt-4o,anthropic/claude-sonnet-4-0",
			want:     true,
		},
		{
			name:     "three inline models",
			modelRef: "openai/gpt-4o,anthropic/claude-sonnet-4-0,google/gemini-2.0-flash",
			want:     true,
		},
		{
			name:     "with spaces",
			modelRef: "openai/gpt-4o, anthropic/claude-sonnet-4-0",
			want:     true,
		},
		{
			name:     "named model (no slash)",
			modelRef: "my_fast_model",
			want:     false,
		},
		{
			name:     "comma separated named models (not inline alloy)",
			modelRef: "fast_model,smart_model",
			want:     false,
		},
		{
			name:     "mixed named and inline",
			modelRef: "fast_model,openai/gpt-4o",
			want:     false, // "fast_model" doesn't contain "/" so it's not an inline alloy
		},
		{
			name:     "empty string",
			modelRef: "",
			want:     false,
		},
		{
			name:     "just commas",
			modelRef: ",,",
			want:     false, // No valid parts after trimming
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isInlineAlloySpec(tt.modelRef)
			assert.Equal(t, tt.want, got)
		})
	}
}
