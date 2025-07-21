package env

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoFailProviderFound(t *testing.T) {
	t.Setenv("TEST1", "VALUE1")

	provider := NewNoFailProvider(NewEnvVariableProvider())
	value, err := provider.GetEnv(t.Context(), "TEST1")

	require.NoError(t, err)
	assert.Equal(t, "VALUE1", value)
}

func TestNoFailProviderNotFound(t *testing.T) {
	t.Setenv("TEST2", "")

	provider := NewNoFailProvider(NewEnvVariableProvider())
	value, err := provider.GetEnv(t.Context(), "TEST2")

	require.NoError(t, err)
	assert.Empty(t, value)
}

func TestNoFailProviderIgnoreError(t *testing.T) {
	provider := NewNoFailProvider(&alwaysFailProvider{})
	value, err := provider.GetEnv(t.Context(), "TEST3")

	require.NoError(t, err)
	assert.Empty(t, value)
}

type alwaysFailProvider struct{}

func (p *alwaysFailProvider) GetEnv(context.Context, string) (string, error) {
	return "Ignored", errors.New("not found")
}
