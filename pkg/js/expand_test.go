package js

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		commands string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "no placeholder",
			commands: "List all files",
			envVars:  map[string]string{},
			expected: "List all files",
		},
		{
			name:     "single placeholder",
			commands: "Say hello to ${env.USER}",
			envVars:  map[string]string{"USER": "alice"},
			expected: "Say hello to alice",
		},
		{
			name:     "multiple placeholders",
			commands: "Analyze ${env.PROJECT_NAME} in ${env.ENVIRONMENT}",
			envVars:  map[string]string{"PROJECT_NAME": "myproject", "ENVIRONMENT": "production"},
			expected: "Analyze myproject in production",
		},
		{
			name:     "missing env var expands to empty string",
			commands: "Check ${env.MISSING_VAR} status",
			envVars:  map[string]string{},
			expected: "Check  status",
		},
		{
			name:     "ternary operator",
			commands: "${env.NAME == 'bob' ? 'Yes' : 'No'}",
			envVars:  map[string]string{"NAME": "bob"},
			expected: "Yes",
		},
		{
			name:     "default value (found)",
			commands: "${env.NAME || 'UNKNOWN'}",
			envVars:  map[string]string{"NAME": "bob"},
			expected: "bob",
		},
		{
			name:     "default value (not found)",
			commands: "${env.NAME || 'UNKNOWN'}",
			envVars:  map[string]string{},
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := testEnvProvider(tt.envVars)

			expander := NewJsExpander(&env)
			result := expander.Expand(t.Context(), tt.commands)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandMap(t *testing.T) {
	t.Parallel()

	env := testEnvProvider(map[string]string{
		"USER": "alice",
	})

	expander := NewJsExpander(&env)
	result := expander.ExpandMap(t.Context(), map[string]string{
		"none":   "List all files",
		"simple": "Say hello to ${env.USER}",
	})

	assert.Equal(t, map[string]string{
		"none":   "List all files",
		"simple": "Say hello to alice",
	}, result)
}

func TestExpandMap_Reuse(t *testing.T) {
	t.Parallel()

	env := testEnvProvider(map[string]string{
		"USER": "alice",
	})

	expander := NewJsExpander(&env)

	result := expander.ExpandMap(t.Context(), map[string]string{
		"none": "List all files",
	})
	assert.Equal(t, map[string]string{
		"none": "List all files",
	}, result)

	result = expander.ExpandMap(t.Context(), map[string]string{
		"simple": "Say hello to ${env.USER}",
	})
	assert.Equal(t, map[string]string{
		"simple": "Say hello to alice",
	}, result)
}

func TestExpandMap_Empty(t *testing.T) {
	t.Parallel()

	env := testEnvProvider(map[string]string{})

	expander := NewJsExpander(&env)
	result := expander.ExpandMap(t.Context(), map[string]string{})

	assert.Empty(t, result)
}

type testEnvProvider map[string]string

func (p *testEnvProvider) Get(_ context.Context, name string) string {
	return (*p)[name]
}
