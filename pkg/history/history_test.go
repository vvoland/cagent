package history

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	h, err := New()
	require.NoError(t, err)

	assert.Equal(t, -1, h.current)
	assert.Empty(t, h.Messages)
}

func TestHistory_AddAndSave(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	h, err := New()
	require.NoError(t, err)

	// Test adding messages
	messages := []string{"first", "second", "third"}
	for _, msg := range messages {
		err := h.Add(msg)
		require.NoError(t, err)
	}

	// Verify messages were added
	assert.Equal(t, messages, h.Messages)
	assert.Len(t, messages, h.current)

	// Test persistence by creating a new instance
	h2, err := New()
	require.NoError(t, err)
	assert.Equal(t, messages, h2.Messages)
}

func TestHistory_Navigation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	h, err := New()
	require.NoError(t, err)

	// Test empty history
	assert.Empty(t, h.Previous())
	assert.Empty(t, h.Next())

	// Add test messages
	messages := []string{"first", "second", "third"}
	for _, msg := range messages {
		require.NoError(t, h.Add(msg))
	}

	// Test Previous() navigation
	assert.Equal(t, "third", h.Previous())
	assert.Equal(t, "second", h.Previous())
	assert.Equal(t, "first", h.Previous())
	// Test staying at beginning
	assert.Equal(t, "first", h.Previous())

	// Test Next() navigation
	assert.Equal(t, "second", h.Next())
	assert.Equal(t, "third", h.Next())
	// Test going past the end
	assert.Empty(t, h.Next())
}

func TestHistory_EdgeCases(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	h, err := New()
	require.NoError(t, err)

	// Test empty history navigation
	assert.Empty(t, h.Previous())
	assert.Empty(t, h.Next())

	// Add single message
	require.NoError(t, h.Add("only"))

	// Test navigation with single message
	assert.Equal(t, "only", h.Previous())
	assert.Equal(t, "only", h.Previous()) // Should stay at the beginning
	assert.Empty(t, h.Next())             // Should return empty when going past the end
}
