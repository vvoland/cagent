package env

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiProviderNone(t *testing.T) {
	provider := NewMultiProvider()
	value, err := provider.GetEnv(t.Context(), "TEST1")

	require.NoError(t, err)
	assert.Empty(t, value)
}

func TestMultiProviderDelegate(t *testing.T) {
	provider := NewMultiProvider(&alwaysFound{}, &neverFound{}, &alwaysFailProvider{})
	value, err := provider.GetEnv(t.Context(), "TEST2")

	require.NoError(t, err)
	assert.Equal(t, "FOUND", value)
}

func TestMultiProviderTryInOrder(t *testing.T) {
	provider := NewMultiProvider(&neverFound{}, &alwaysFound{}, &alwaysFailProvider{})
	value, err := provider.GetEnv(t.Context(), "TEST3")

	require.NoError(t, err)
	assert.Equal(t, "FOUND", value)
}

func TestMultiProviderFails(t *testing.T) {
	provider := NewMultiProvider(&alwaysFailProvider{})
	value, err := provider.GetEnv(t.Context(), "TEST4")

	require.Error(t, err)
	assert.Empty(t, value)
}

type neverFound struct{}

func (p *neverFound) GetEnv(context.Context, string) (string, error) {
	return "", nil
}

type alwaysFound struct{}

func (p *alwaysFound) GetEnv(context.Context, string) (string, error) {
	return "FOUND", nil
}
