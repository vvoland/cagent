package gemini

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGemini3PlusModel(t *testing.T) {
	t.Parallel()

	match := []string{
		"gemini-3-pro", "gemini-3-pro-preview",
		"gemini-3-flash", "gemini-3-flash-preview",
		"gemini-3.1-pro-preview", "gemini-3.1-flash-preview",
		"gemini-3.5-pro", "gemini-3.5-flash",
	}
	noMatch := []string{
		"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash",
		"gemini-1.5-pro", "gpt-4o", "claude-sonnet-4-0",
		"gemini-3",      // no trailing separator
		"gemini-30-pro", // "0" ≠ '-' or '.'
		"gemini-3.",     // dot with no version digit or dash
		"gemini-3.1",    // dot-version but no trailing dash
	}

	for _, m := range match {
		assert.True(t, isGemini3PlusModel(m), "%q should match", m)
	}
	for _, m := range noMatch {
		assert.False(t, isGemini3PlusModel(m), "%q should not match", m)
	}
}
