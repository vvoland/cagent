package modelerrors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	openai "github.com/openai/openai-go/v3"
	"github.com/stretchr/testify/assert"
)

// mockTimeoutError implements net.Error with Timeout() = true
type mockTimeoutError struct{}

func (e *mockTimeoutError) Error() string   { return "mock timeout" }
func (e *mockTimeoutError) Timeout() bool   { return true }
func (e *mockTimeoutError) Temporary() bool { return true }

var _ net.Error = (*mockTimeoutError)(nil)

func TestIsRetryableModelError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "nil error", err: nil, expected: false},
		{name: "context canceled", err: context.Canceled, expected: false},
		{name: "context deadline exceeded", err: context.DeadlineExceeded, expected: false},
		{name: "network timeout", err: &mockTimeoutError{}, expected: true},
		{name: "rate limit 429", err: errors.New("API error: status 429 too many requests"), expected: false},
		{name: "rate limit message", err: errors.New("rate limit exceeded"), expected: false},
		{name: "too many requests", err: errors.New("too many requests"), expected: false},
		{name: "throttling", err: errors.New("request throttled"), expected: false},
		{name: "quota exceeded", err: errors.New("quota exceeded"), expected: false},
		{name: "server error 500", err: errors.New("internal server error 500"), expected: true},
		{name: "bad gateway 502", err: errors.New("502 bad gateway"), expected: true},
		{name: "service unavailable 503", err: errors.New("503 service unavailable"), expected: true},
		{name: "gateway timeout 504", err: errors.New("504 gateway timeout"), expected: true},
		{name: "timeout message", err: errors.New("request timeout"), expected: true},
		{name: "connection refused", err: errors.New("connection refused"), expected: true},
		{name: "unauthorized 401", err: errors.New("401 unauthorized"), expected: false},
		{name: "forbidden 403", err: errors.New("403 forbidden"), expected: false},
		{name: "not found 404", err: errors.New("404 not found"), expected: false},
		{name: "bad request 400", err: errors.New("400 bad request"), expected: false},
		{name: "api key error", err: errors.New("invalid api key"), expected: false},
		{name: "authentication error", err: errors.New("authentication failed"), expected: false},
		{name: "anthropic overloaded 529", err: errors.New("529 overloaded"), expected: true},
		{name: "other side closed", err: errors.New("other side closed the connection"), expected: true},
		{name: "fetch failed", err: errors.New("fetch failed"), expected: true},
		{name: "reset before headers", err: errors.New("reset before headers"), expected: true},
		{name: "upstream connect error", err: errors.New("upstream connect error"), expected: true},
		{name: "HTTP/2 INTERNAL_ERROR", err: fmt.Errorf("error receiving from stream: %w", errors.New("stream error: stream ID 1; INTERNAL_ERROR; received from peer")), expected: true},
		{name: "context overflow - prompt too long", err: errors.New("prompt is too long: 226360 tokens > 200000 maximum"), expected: false},
		{name: "context overflow - thinking budget", err: errors.New("max_tokens must be greater than thinking.budget_tokens"), expected: false},
		{name: "context overflow - wrapped", err: &ContextOverflowError{Underlying: errors.New("test")}, expected: false},
		{name: "unknown error", err: errors.New("something weird happened"), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsRetryableModelError(tt.err), "IsRetryableModelError(%v)", tt.err)
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{attempt: 0, minExpected: 180 * time.Millisecond, maxExpected: 220 * time.Millisecond},
		{attempt: 1, minExpected: 360 * time.Millisecond, maxExpected: 440 * time.Millisecond},
		{attempt: 2, minExpected: 720 * time.Millisecond, maxExpected: 880 * time.Millisecond},
		{attempt: 3, minExpected: 1440 * time.Millisecond, maxExpected: 1760 * time.Millisecond},
		{attempt: 10, minExpected: 1800 * time.Millisecond, maxExpected: 2200 * time.Millisecond}, // capped at 2s
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			t.Parallel()
			backoff := CalculateBackoff(tt.attempt)
			assert.GreaterOrEqual(t, backoff, tt.minExpected, "backoff should be at least %v", tt.minExpected)
			assert.LessOrEqual(t, backoff, tt.maxExpected, "backoff should be at most %v", tt.maxExpected)
		})
	}

	t.Run("negative attempt treated as 0", func(t *testing.T) {
		t.Parallel()
		backoff := CalculateBackoff(-1)
		assert.GreaterOrEqual(t, backoff, 180*time.Millisecond)
		assert.LessOrEqual(t, backoff, 220*time.Millisecond)
	})
}

func TestSleepWithContext(t *testing.T) {
	t.Parallel()

	t.Run("completes normally", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		start := time.Now()
		completed := SleepWithContext(ctx, 10*time.Millisecond)
		elapsed := time.Since(start)

		assert.True(t, completed, "should complete normally")
		assert.GreaterOrEqual(t, elapsed, 10*time.Millisecond)
	})

	t.Run("interrupted by context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		time.AfterFunc(10*time.Millisecond, cancel)

		start := time.Now()
		completed := SleepWithContext(ctx, 1*time.Second)
		elapsed := time.Since(start)

		assert.False(t, completed, "should be interrupted")
		assert.Less(t, elapsed, 100*time.Millisecond, "should return quickly after cancel")
	})
}

func TestExtractHTTPStatusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "nil error", err: nil, expected: 0},
		{name: "429 in message", err: errors.New("POST /v1/chat/completions: 429 Too Many Requests"), expected: 429},
		{name: "500 in message", err: errors.New("internal server error 500"), expected: 500},
		{name: "502 in message", err: errors.New("502 bad gateway"), expected: 502},
		{name: "401 in message", err: errors.New("401 unauthorized"), expected: 401},
		{name: "no status code", err: errors.New("connection refused"), expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ExtractHTTPStatusCode(tt.err), "ExtractHTTPStatusCode(%v)", tt.err)
		})
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		statusCode int
		expected   bool
	}{
		{500, true}, {502, true}, {503, true}, {504, true}, // Server errors
		{408, true},                                            // Request timeout
		{529, true},                                            // Anthropic overloaded
		{429, false},                                           // Rate limit
		{400, false}, {401, false}, {403, false}, {404, false}, // Client errors
		{200, false}, // Not an error
		{0, false},   // Unknown
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsRetryableStatusCode(tt.statusCode), "IsRetryableStatusCode(%d)", tt.statusCode)
		})
	}
}

func TestIsContextOverflowError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "nil error", err: nil, expected: false},
		{name: "generic error", err: errors.New("something went wrong"), expected: false},
		{name: "anthropic prompt too long", err: errors.New("prompt is too long: 226360 tokens > 200000 maximum"), expected: true},
		{name: "openai context length exceeded", err: errors.New("maximum context length is 128000 tokens"), expected: true},
		{name: "context_length_exceeded code", err: errors.New("error code: context_length_exceeded"), expected: true},
		{name: "thinking budget error", err: errors.New("max_tokens must be greater than thinking.budget_tokens"), expected: true},
		{name: "request too large", err: errors.New("request too large for model"), expected: true},
		{name: "input is too long", err: errors.New("input is too long"), expected: true},
		{name: "reduce your prompt", err: errors.New("please reduce your prompt"), expected: true},
		{name: "reduce the length", err: errors.New("please reduce the length of the messages"), expected: true},
		{name: "token limit", err: errors.New("token limit exceeded"), expected: true},
		{name: "wrapped ContextOverflowError", err: &ContextOverflowError{Underlying: errors.New("test")}, expected: true},
		{name: "errors.As wrapped", err: fmt.Errorf("all models failed: %w", &ContextOverflowError{Underlying: errors.New("test")}), expected: true},
		{name: "500 internal server error", err: errors.New("500 Internal Server Error"), expected: false},
		{name: "429 rate limit", err: errors.New("429 too many requests"), expected: false},
		{name: "network timeout", err: errors.New("connection timeout"), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsContextOverflowError(tt.err), "IsContextOverflowError(%v)", tt.err)
		})
	}
}

func TestContextOverflowError(t *testing.T) {
	t.Parallel()

	t.Run("wraps underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("prompt is too long: 226360 tokens > 200000 maximum")
		ctxErr := &ContextOverflowError{Underlying: underlying}

		assert.Contains(t, ctxErr.Error(), "context window overflow")
		assert.Contains(t, ctxErr.Error(), "prompt is too long")
		assert.ErrorIs(t, ctxErr, underlying)
	})

	t.Run("errors.As works", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("test error")
		wrapped := fmt.Errorf("all models failed: %w", &ContextOverflowError{Underlying: underlying})

		var ctxErr *ContextOverflowError
		assert.ErrorAs(t, wrapped, &ctxErr)
	})
}

func TestIsRetryableModelError_ContextOverflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "prompt too long", err: errors.New("prompt is too long: 226360 tokens > 200000 maximum")},
		{name: "thinking budget cascade", err: errors.New("max_tokens must be greater than thinking.budget_tokens")},
		{name: "context length exceeded", err: errors.New("maximum context length is 128000 tokens")},
		{name: "wrapped ContextOverflowError", err: &ContextOverflowError{Underlying: errors.New("test")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsRetryableModelError(tt.err), "context overflow errors should not be retryable: %v", tt.err)
		})
	}
}

func TestFormatError(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, FormatError(nil))
	})

	t.Run("context overflow shows user-friendly message", func(t *testing.T) {
		t.Parallel()
		err := &ContextOverflowError{Underlying: errors.New("prompt is too long")}
		msg := FormatError(err)
		assert.Contains(t, msg, "context window")
		assert.Contains(t, msg, "/compact")
		assert.NotContains(t, msg, "prompt is too long")
	})

	t.Run("generic error preserves message", func(t *testing.T) {
		t.Parallel()
		err := errors.New("authentication failed")
		assert.Equal(t, "authentication failed", FormatError(err))
	})
}

// makeAnthropicError creates an *anthropic.Error with the given status code and
// optional Retry-After header value. Used for testing ExtractRetryAfter.
func makeAnthropicError(statusCode int, retryAfterValue string) *anthropic.Error {
	header := http.Header{}
	if retryAfterValue != "" {
		header.Set("Retry-After", retryAfterValue)
	}
	resp := httptest.NewRecorder().Result()
	resp.StatusCode = statusCode
	resp.Header = header
	// anthropic.Error.Error() dereferences Request, so we must provide a non-nil one.
	req, _ := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", http.NoBody)
	return &anthropic.Error{
		StatusCode: statusCode,
		Response:   resp,
		Request:    req,
	}
}

// makeOpenAIError creates an *openai.Error with the given status code and
// optional Retry-After header value. Used for testing ExtractRetryAfter.
func makeOpenAIError(statusCode int, retryAfterValue string) *openai.Error {
	header := http.Header{}
	if retryAfterValue != "" {
		header.Set("Retry-After", retryAfterValue)
	}
	resp := httptest.NewRecorder().Result()
	resp.StatusCode = statusCode
	resp.Header = header
	// openai.Error.Error() dereferences Request, so we must provide a non-nil one.
	req, _ := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", http.NoBody)
	return &openai.Error{
		StatusCode: statusCode,
		Response:   resp,
		Request:    req,
	}
}

func TestParseRetryAfterHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{name: "empty", value: "", expected: 0},
		{name: "zero seconds", value: "0", expected: 0},
		{name: "negative seconds", value: "-1", expected: 0},
		{name: "invalid string", value: "foo", expected: 0},
		{name: "5 seconds", value: "5", expected: 5 * time.Second},
		{name: "30 seconds", value: "30", expected: 30 * time.Second},
		{name: "120 seconds", value: "120", expected: 120 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseRetryAfterHeader(tt.value)
			assert.Equal(t, tt.expected, got, "parseRetryAfterHeader(%q)", tt.value)
		})
	}

	t.Run("HTTP-date in the future", func(t *testing.T) {
		t.Parallel()
		// Use a time 10 seconds in the future
		future := time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat)
		got := parseRetryAfterHeader(future)
		assert.Greater(t, got, 0*time.Second, "should return positive duration for future HTTP-date")
		assert.LessOrEqual(t, got, 11*time.Second, "should not exceed ~10s for near-future date")
	})

	t.Run("HTTP-date in the past", func(t *testing.T) {
		t.Parallel()
		past := time.Now().Add(-10 * time.Second).UTC().Format(http.TimeFormat)
		got := parseRetryAfterHeader(past)
		assert.Equal(t, 0*time.Second, got, "should return 0 for past HTTP-date")
	})
}

func TestExtractRetryAfter(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns 0", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, time.Duration(0), ExtractRetryAfter(nil))
	})

	t.Run("plain error returns 0", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, time.Duration(0), ExtractRetryAfter(errors.New("some error")))
	})

	t.Run("anthropic error with Retry-After seconds", func(t *testing.T) {
		t.Parallel()
		err := makeAnthropicError(429, "15")
		assert.Equal(t, 15*time.Second, ExtractRetryAfter(err))
	})

	t.Run("anthropic error without Retry-After header", func(t *testing.T) {
		t.Parallel()
		err := makeAnthropicError(429, "")
		assert.Equal(t, time.Duration(0), ExtractRetryAfter(err))
	})

	t.Run("openai error with Retry-After seconds", func(t *testing.T) {
		t.Parallel()
		err := makeOpenAIError(429, "30")
		assert.Equal(t, 30*time.Second, ExtractRetryAfter(err))
	})

	t.Run("openai error without Retry-After header", func(t *testing.T) {
		t.Parallel()
		err := makeOpenAIError(429, "")
		assert.Equal(t, time.Duration(0), ExtractRetryAfter(err))
	})

	t.Run("wrapped anthropic error", func(t *testing.T) {
		t.Parallel()
		anthropicErr := makeAnthropicError(429, "5")
		wrapped := fmt.Errorf("model failed: %w", anthropicErr)
		assert.Equal(t, 5*time.Second, ExtractRetryAfter(wrapped))
	})
}

func TestClassifyModelError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		err              error
		wantRetryable    bool
		wantRateLimited  bool
		wantRetryAfterGT time.Duration // retryAfter should be > this (0 means just checking it's >=0)
	}{
		{name: "nil", err: nil, wantRetryable: false, wantRateLimited: false},
		{name: "context canceled", err: context.Canceled, wantRetryable: false, wantRateLimited: false},
		{name: "context deadline exceeded", err: context.DeadlineExceeded, wantRetryable: false, wantRateLimited: false},
		{name: "context overflow", err: errors.New("prompt is too long: 200000 tokens > 100000 maximum"), wantRetryable: false, wantRateLimited: false},
		// 429 rate limit cases
		{name: "429 message only", err: errors.New("POST /v1/chat: 429 Too Many Requests"), wantRetryable: false, wantRateLimited: true},
		{name: "429 anthropic error no header", err: makeAnthropicError(429, ""), wantRetryable: false, wantRateLimited: true},
		{name: "429 openai error no header", err: makeOpenAIError(429, ""), wantRetryable: false, wantRateLimited: true},
		{name: "500 openai error", err: makeOpenAIError(500, ""), wantRetryable: true, wantRateLimited: false},
		// Retryable server errors
		{name: "500 message", err: errors.New("500 internal server error"), wantRetryable: true, wantRateLimited: false},
		{name: "500 anthropic error", err: makeAnthropicError(500, ""), wantRetryable: true, wantRateLimited: false},
		{name: "502 bad gateway", err: errors.New("502 bad gateway"), wantRetryable: true, wantRateLimited: false},
		{name: "503 service unavailable", err: errors.New("503 service unavailable"), wantRetryable: true, wantRateLimited: false},
		{name: "504 gateway timeout", err: errors.New("504 gateway timeout"), wantRetryable: true, wantRateLimited: false},
		{name: "529 overloaded", err: makeAnthropicError(529, ""), wantRetryable: true, wantRateLimited: false},
		{name: "408 timeout", err: makeAnthropicError(408, ""), wantRetryable: true, wantRateLimited: false},
		// Non-retryable client errors
		{name: "400 bad request", err: makeAnthropicError(400, ""), wantRetryable: false, wantRateLimited: false},
		{name: "401 unauthorized", err: makeAnthropicError(401, ""), wantRetryable: false, wantRateLimited: false},
		{name: "403 forbidden", err: makeAnthropicError(403, ""), wantRetryable: false, wantRateLimited: false},
		// Network errors
		{name: "network timeout", err: &mockTimeoutError{}, wantRetryable: true, wantRateLimited: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			retryable, rateLimited, retryAfter := ClassifyModelError(tt.err)
			assert.Equal(t, tt.wantRetryable, retryable, "retryable mismatch")
			assert.Equal(t, tt.wantRateLimited, rateLimited, "rateLimited mismatch")
			assert.GreaterOrEqual(t, retryAfter, time.Duration(0), "retryAfter should never be negative")
		})
	}

	t.Run("429 with Retry-After header propagated", func(t *testing.T) {
		t.Parallel()
		err := makeAnthropicError(429, "20")
		retryable, rateLimited, retryAfter := ClassifyModelError(err)
		assert.False(t, retryable)
		assert.True(t, rateLimited)
		assert.Equal(t, 20*time.Second, retryAfter)
	})

	t.Run("429 openai with Retry-After header", func(t *testing.T) {
		t.Parallel()
		err := makeOpenAIError(429, "10")
		retryable, rateLimited, retryAfter := ClassifyModelError(err)
		assert.False(t, retryable)
		assert.True(t, rateLimited)
		assert.Equal(t, 10*time.Second, retryAfter)
	})
}
