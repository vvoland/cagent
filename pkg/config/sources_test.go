package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLSource_Read(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test content"))
	}))
	t.Cleanup(server.Close)

	source := NewURLSource(server.URL)

	assert.Equal(t, server.URL, source.Name())
	assert.Empty(t, source.ParentDir())

	data, err := source.Read(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "test content", string(data))
}

func TestURLSource_Read_HTTPError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
	}{
		{"not found", http.StatusNotFound},
		{"server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			t.Cleanup(server.Close)

			_, err := NewURLSource(server.URL).Read(t.Context())
			require.Error(t, err)
		})
	}
}

func TestURLSource_Read_ConnectionError(t *testing.T) {
	t.Parallel()

	_, err := NewURLSource("http://invalid.invalid/config.yaml").Read(t.Context())
	require.Error(t, err)
}

func TestIsURLReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected bool
	}{
		{"http://example.com/agent.yaml", true},
		{"https://example.com/agent.yaml", true},
		{"https://example.com:8080/path", true},
		{"/path/to/agent.yaml", false},
		{"./agent.yaml", false},
		{"docker.io/myorg/agent:v1", false},
		{"ftp://example.com/agent.yaml", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsURLReference(tt.input))
		})
	}
}

func TestResolve_URLReference(t *testing.T) {
	t.Parallel()

	source, err := Resolve("https://example.com/agent.yaml")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/agent.yaml", source.Name())
	assert.Empty(t, source.ParentDir())
}

func TestResolveSources_URLReference(t *testing.T) {
	t.Parallel()

	url := "https://example.com/agent.yaml"
	sources, err := ResolveSources(url)
	require.NoError(t, err)
	require.Len(t, sources, 1)

	source, ok := sources[url]
	require.True(t, ok)
	assert.Equal(t, url, source.Name())
}
