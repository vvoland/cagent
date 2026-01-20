package fake

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
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

// slowReader is a reader that blocks until the context is canceled or data is written
type slowReader struct {
	data   chan []byte
	closed chan struct{}
}

func newSlowReader() *slowReader {
	return &slowReader{
		data:   make(chan []byte, 1),
		closed: make(chan struct{}),
	}
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	select {
	case data := <-r.data:
		return copy(p, data), nil
	case <-r.closed:
		return 0, io.EOF
	}
}

func (r *slowReader) Close() error {
	close(r.closed)
	return nil
}

// readerFromRecorder wraps httptest.ResponseRecorder to implement io.ReaderFrom
type readerFromRecorder struct {
	*httptest.ResponseRecorder
}

func (r *readerFromRecorder) ReadFrom(src io.Reader) (n int64, err error) {
	return io.Copy(r.ResponseRecorder, src)
}

func TestStreamCopy_ContextCancellation(t *testing.T) {
	// Create a slow reader that blocks until closed
	slowBody := newSlowReader()

	// Create a mock HTTP response with the slow reader
	resp := &http.Response{
		Body: slowBody,
	}

	// Create an echo context with a request that has a cancelable context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := &readerFromRecorder{httptest.NewRecorder()}
	ctx, cancel := context.WithCancel(t.Context())
	req = req.WithContext(ctx)
	c := e.NewContext(req, rec)

	// Start StreamCopy in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- StreamCopy(c, resp)
	}()

	// Give StreamCopy time to start and block on the read
	time.Sleep(50 * time.Millisecond)

	// Cancel the context - this should cause StreamCopy to return immediately
	cancel()

	// StreamCopy should return within a reasonable time
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("StreamCopy did not return after context cancellation")
	}
}

func TestStreamCopy_NormalCompletion(t *testing.T) {
	// Create a response with a normal body
	body := bytes.NewReader([]byte("test data"))
	resp := &http.Response{
		Body: io.NopCloser(body),
	}

	// Create an echo context with a wrapped recorder
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := &readerFromRecorder{httptest.NewRecorder()}
	c := e.NewContext(req, rec)

	// StreamCopy should complete successfully
	err := StreamCopy(c, resp)
	require.NoError(t, err)

	// Verify the data was written
	assert.Equal(t, "test data", rec.Body.String())
}
