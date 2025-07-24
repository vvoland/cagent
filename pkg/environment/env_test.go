package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOsEnvProviderFound(t *testing.T) {
	t.Setenv("TEST1", "VALUE1")

	provider := NewOsEnvProvider()
	value, err := provider.Get(t.Context(), "TEST1")

	require.NoError(t, err)
	assert.Equal(t, "VALUE1", value)
}

func TestOsEnvProviderNotFound(t *testing.T) {
	t.Setenv("TEST2", "")

	provider := NewOsEnvProvider()
	value, err := provider.Get(t.Context(), "TEST2")

	require.NoError(t, err)
	assert.Empty(t, value)
}
