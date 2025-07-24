package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOsEnvProvider(t *testing.T) {
	t.Setenv("TEST1", "VALUE1")
	t.Setenv("TEST2", "VALUE2")

	provider := NewOsEnvProvider()

	value, err := provider.Get(t.Context(), "TEST1")
	require.NoError(t, err)
	assert.Equal(t, "VALUE1", value)

	value, err = provider.Get(t.Context(), "TEST2")
	require.NoError(t, err)
	assert.Equal(t, "VALUE2", value)

	value, err = provider.Get(t.Context(), "NOT_FOUND")
	require.NoError(t, err)
	assert.Empty(t, value)
}

func TestKeyValueProvider(t *testing.T) {
	t.Setenv("OS_ENV", "VALUE3")

	provider := NewKeyValueProvider(map[string]string{
		"TEST1": "VALUE1",
		"TEST2": "VALUE2",
		"TEST3": "$OS_ENV",
	})

	value, err := provider.Get(t.Context(), "TEST1")
	require.NoError(t, err)
	assert.Equal(t, "VALUE1", value)

	value, err = provider.Get(t.Context(), "TEST2")
	require.NoError(t, err)
	assert.Equal(t, "VALUE2", value)

	value, err = provider.Get(t.Context(), "TEST3")
	require.NoError(t, err)
	assert.Equal(t, "VALUE3", value)

	value, err = provider.Get(t.Context(), "NOT_FOUND")
	require.NoError(t, err)
	assert.Empty(t, value)
}
