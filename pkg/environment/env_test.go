package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOsEnvProvider(t *testing.T) {
	t.Setenv("TEST1", "VALUE1")
	t.Setenv("TEST2", "VALUE2")

	provider := NewOsEnvProvider()

	value := provider.Get(t.Context(), "TEST1")
	assert.Equal(t, "VALUE1", value)

	value = provider.Get(t.Context(), "TEST2")
	assert.Equal(t, "VALUE2", value)

	value = provider.Get(t.Context(), "NOT_FOUND")
	assert.Empty(t, value)
}

func TestNewEnvListProvider(t *testing.T) {
	t.Parallel()

	provider := NewEnvListProvider([]string{
		"TEST1=VALUE1",
		"TEST2=VALUE2",
	})

	value := provider.Get(t.Context(), "TEST1")
	assert.Equal(t, "VALUE1", value)

	value = provider.Get(t.Context(), "TEST2")
	assert.Equal(t, "VALUE2", value)

	value = provider.Get(t.Context(), "NOT_FOUND")
	assert.Empty(t, value)
}
