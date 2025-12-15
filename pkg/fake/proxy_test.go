package fake

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyHeaderUpdater(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		envKey         string
		envValue       string
		expectedHeader string
		expectedValue  string
	}{
		{
			name:           "OpenAI",
			host:           "https://api.openai.com/v1",
			envKey:         "OPENAI_API_KEY",
			envValue:       "test-openai-key",
			expectedHeader: "Authorization",
			expectedValue:  "Bearer test-openai-key",
		},
		{
			name:           "Anthropic",
			host:           "https://api.anthropic.com",
			envKey:         "ANTHROPIC_API_KEY",
			envValue:       "test-anthropic-key",
			expectedHeader: "X-Api-Key",
			expectedValue:  "test-anthropic-key",
		},
		{
			name:           "Google",
			host:           "https://generativelanguage.googleapis.com",
			envKey:         "GOOGLE_API_KEY",
			envValue:       "test-google-key",
			expectedHeader: "X-Goog-Api-Key",
			expectedValue:  "test-google-key",
		},
		{
			name:           "Mistral",
			host:           "https://api.mistral.ai/v1",
			envKey:         "MISTRAL_API_KEY",
			envValue:       "test-mistral-key",
			expectedHeader: "Authorization",
			expectedValue:  "Bearer test-mistral-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envValue)

			req, err := http.NewRequest(http.MethodPost, "https://example.com", http.NoBody)
			require.NoError(t, err)

			APIKeyHeaderUpdater(tt.host, req)

			assert.Equal(t, tt.expectedValue, req.Header.Get(tt.expectedHeader))
		})
	}
}

func TestAPIKeyHeaderUpdater_UnknownHost(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com", http.NoBody)
	require.NoError(t, err)

	APIKeyHeaderUpdater("https://unknown.host.com", req)

	assert.Empty(t, req.Header.Get("Authorization"))
	assert.Empty(t, req.Header.Get("X-Api-Key"))
}

func TestTargetURLForHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host     string
		expected bool
	}{
		{"https://api.openai.com/v1", true},
		{"https://api.anthropic.com", true},
		{"https://generativelanguage.googleapis.com", true},
		{"https://api.mistral.ai/v1", true},
		{"https://unknown.host.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			t.Parallel()

			fn := TargetURLForHost(tt.host)
			if tt.expected {
				assert.NotNil(t, fn)
			} else {
				assert.Nil(t, fn)
			}
		})
	}
}
