package environment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiProviderNone(t *testing.T) {
	provider := NewMultiProvider()
	value := provider.Get(t.Context(), "TEST1")

	assert.Empty(t, value)
}

func TestMultiProviderDelegate(t *testing.T) {
	provider := NewMultiProvider(&alwaysFound{}, &neverFound{})
	value := provider.Get(t.Context(), "TEST2")

	assert.Equal(t, "FOUND", value)
}

func TestMultiProviderTryInOrder(t *testing.T) {
	provider := NewMultiProvider(&neverFound{}, &alwaysFound{})
	value := provider.Get(t.Context(), "TEST3")

	assert.Equal(t, "FOUND", value)
}

type neverFound struct{}

func (p *neverFound) Get(context.Context, string) string {
	return ""
}

type alwaysFound struct{}

func (p *alwaysFound) Get(context.Context, string) string {
	return "FOUND"
}
