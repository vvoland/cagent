package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvVariableProviderFound(t *testing.T) {
	t.Setenv("TEST1", "VALUE1")

	provider := NewEnvVariableProvider()
	value, err := provider.GetEnv(t.Context(), "TEST1")

	require.NoError(t, err)
	assert.Equal(t, "VALUE1", value)
}

func TestEnvVariableProviderNotFound(t *testing.T) {
	t.Setenv("TEST2", "")

	provider := NewEnvVariableProvider()
	value, err := provider.GetEnv(t.Context(), "TEST2")

	require.NoError(t, err)
	assert.Empty(t, value)
}
