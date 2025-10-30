package js

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	tests := []struct {
		name     string
		commands map[string]string
		envVars  map[string]string
		expected map[string]string
	}{
		{
			name:     "no placeholder",
			commands: map[string]string{"simple": "List all files"},
			envVars:  map[string]string{},
			expected: map[string]string{"simple": "List all files"},
		},
		{
			name:     "single placeholder",
			commands: map[string]string{"greet": "Say hello to ${env.USER}"},
			envVars:  map[string]string{"USER": "alice"},
			expected: map[string]string{"greet": "Say hello to alice"},
		},
		{
			name:     "multiple placeholders",
			commands: map[string]string{"analyze": "Analyze ${env.PROJECT_NAME} in ${env.ENVIRONMENT}"},
			envVars:  map[string]string{"PROJECT_NAME": "myproject", "ENVIRONMENT": "production"},
			expected: map[string]string{"analyze": "Analyze myproject in production"},
		},
		{
			name:     "missing env var expands to empty string",
			commands: map[string]string{"test": "Check ${env.MISSING_VAR} status"},
			envVars:  map[string]string{},
			expected: map[string]string{"test": "Check  status"},
		},
		{
			name:     "ternary operator",
			commands: map[string]string{"test": "${env.NAME == 'bob' ? 'Yes' : 'No'}"},
			envVars:  map[string]string{"NAME": "bob"},
			expected: map[string]string{"test": "Yes"},
		},
		{
			name:     "default value (found)",
			commands: map[string]string{"test": "${env.NAME || 'UNKNOWN'}"},
			envVars:  map[string]string{"NAME": "bob"},
			expected: map[string]string{"test": "bob"},
		},
		{
			name:     "default value (not found)",
			commands: map[string]string{"test": "${env.NAME || 'UNKNOWN'}"},
			envVars:  map[string]string{},
			expected: map[string]string{"test": "UNKNOWN"},
		},
		{
			name:     "empty commands",
			commands: map[string]string{},
			envVars:  map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := testEnvProvider(tt.envVars)

			result := Expand(t.Context(), tt.commands, &env)

			assert.Equal(t, tt.expected, result)
		})
	}
}

type testEnvProvider map[string]string

func (p *testEnvProvider) Get(_ context.Context, name string) string {
	return (*p)[name]
}
