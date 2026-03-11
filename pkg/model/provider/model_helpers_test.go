package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGemini3Family(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model  string
		family string
	}{
		// Gemini 3 models
		{"gemini-3-pro", "pro"},
		{"gemini-3-pro-preview", "pro-preview"},
		{"gemini-3-flash", "flash"},
		{"gemini-3-flash-preview", "flash-preview"},

		// Gemini 3.x models
		{"gemini-3.1-pro-preview", "pro-preview"},
		{"gemini-3.1-flash-preview", "flash-preview"},
		{"gemini-3.5-pro", "pro"},
		{"gemini-3.5-flash", "flash"},

		// Non-matching models → empty family
		{"gemini-2.5-flash", ""},
		{"gemini-2.5-pro", ""},
		{"gemini-2.0-flash", ""},
		{"gemini-1.5-pro", ""},
		{"gpt-4o", ""},
		{"gemini-3", ""},      // No trailing separator
		{"gemini-30-pro", ""}, // "0" is not '.' or '-'
		{"gemini-3.", ""},     // dot with no version digit or dash
		{"gemini-3.1", ""},    // dot-version but no trailing dash
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.family, gemini3Family(tt.model))
		})
	}
}

func TestIsGeminiProModel(t *testing.T) {
	t.Parallel()

	pro := []string{"gemini-3-pro", "gemini-3-pro-preview", "gemini-3.1-pro-preview", "gemini-3.5-pro"}
	notPro := []string{"gemini-3-flash", "gemini-3.1-flash-preview", "gemini-2.5-pro", "gpt-4o"}

	for _, m := range pro {
		assert.True(t, isGeminiProModel(m), "%q should be pro", m)
	}
	for _, m := range notPro {
		assert.False(t, isGeminiProModel(m), "%q should not be pro", m)
	}
}

func TestIsGeminiFlashModel(t *testing.T) {
	t.Parallel()

	flash := []string{"gemini-3-flash", "gemini-3-flash-preview", "gemini-3.1-flash-preview", "gemini-3.5-flash"}
	notFlash := []string{"gemini-3-pro", "gemini-3.1-pro-preview", "gemini-2.5-flash", "gpt-4o"}

	for _, m := range flash {
		assert.True(t, isGeminiFlashModel(m), "%q should be flash", m)
	}
	for _, m := range notFlash {
		assert.False(t, isGeminiFlashModel(m), "%q should not be flash", m)
	}
}
