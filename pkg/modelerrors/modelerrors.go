// Package modelerrors provides error classification utilities for LLM model
// providers. It determines whether errors are retryable, identifies context
// window overflow conditions, extracts HTTP status codes from various SDK
// error types, and computes exponential backoff durations.
package modelerrors

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Backoff and retry-after configuration constants.
const (
	backoffBaseDelay = 200 * time.Millisecond
	backoffMaxDelay  = 2 * time.Second
	backoffFactor    = 2.0
	backoffJitter    = 0.1

	// MaxRetryAfterWait caps how long we'll honor a Retry-After header to prevent
	// a misbehaving server from blocking the agent for an unreasonable amount of time.
	MaxRetryAfterWait = 60 * time.Second
)

// StatusError wraps an HTTP API error with structured metadata for retry decisions.
// Providers wrap SDK errors in this type so the retry loop can use errors.As
// to extract status code and Retry-After without importing provider-specific SDKs.
type StatusError struct {
	// StatusCode is the HTTP status code from the provider's API response.
	StatusCode int
	// RetryAfter is the parsed Retry-After header duration. Zero if absent.
	RetryAfter time.Duration
	// Err is the original error from the provider SDK.
	Err error
}

func (e *StatusError) Error() string {
	return e.Err.Error()
}

func (e *StatusError) Unwrap() error {
	return e.Err
}

// WrapHTTPError wraps err in a *StatusError carrying the HTTP status code and
// parsed Retry-After header from resp. Returns err unchanged if statusCode < 400
// or err is nil. Pass resp=nil when no *http.Response is available.
func WrapHTTPError(statusCode int, resp *http.Response, err error) error {
	if err == nil || statusCode < 400 {
		return err
	}
	var retryAfter time.Duration
	if resp != nil {
		retryAfter = ParseRetryAfterHeader(resp.Header.Get("Retry-After"))
	}
	return &StatusError{
		StatusCode: statusCode,
		RetryAfter: retryAfter,
		Err:        err,
	}
}

// Default fallback configuration.
const (
	// DefaultRetries is the default number of retries per model with exponential
	// backoff for retryable errors (5xx, timeouts). 2 retries means 3 total attempts.
	// This handles transient provider issues without immediately failing over.
	DefaultRetries = 2

	// DefaultCooldown is the default duration to stick with a fallback model
	// after a non-retryable error before retrying the primary.
	DefaultCooldown = 1 * time.Minute
)

// ContextOverflowError wraps an underlying error to indicate that the failure
// was caused by the conversation context exceeding the model's context window.
// This is used to trigger auto-compaction in the runtime loop instead of
// surfacing raw HTTP errors to the user.
type ContextOverflowError struct {
	Underlying error
}

func (e *ContextOverflowError) Error() string {
	if e.Underlying == nil {
		return "context window overflow"
	}
	return "context window overflow: " + e.Underlying.Error()
}

func (e *ContextOverflowError) Unwrap() error {
	return e.Underlying
}

// contextOverflowPatterns contains error message substrings that indicate the
// prompt/context exceeds the model's context window. These patterns are checked
// case-insensitively against error messages from various providers.
var contextOverflowPatterns = []string{
	"prompt is too long",
	"maximum context length",
	"context length exceeded",
	"context_length_exceeded",
	"max_tokens must be greater than",
	"maximum number of tokens",
	"content length exceeds",
	"request too large",
	"payload too large",
	"input is too long",
	"exceeds the model's max token",
	"token limit",
	"reduce your prompt",
	"reduce the length",
}

// IsContextOverflowError checks whether the error indicates the conversation
// context has exceeded the model's context window. It inspects both structured
// SDK error types and raw error message patterns.
//
// Recognised patterns include:
//   - Anthropic 400 "prompt is too long: N tokens > M maximum"
//   - Anthropic 400 "max_tokens must be greater than thinking.budget_tokens"
//     (emitted when the prompt is so large that max_tokens can't accommodate
//     the thinking budget — a proxy for context overflow)
//   - OpenAI 400 "maximum context length" / "context_length_exceeded"
//   - Anthropic 500 that is actually a context overflow (heuristic: the error
//     message is opaque but the conversation was already near the limit)
//
// This function intentionally does NOT match generic 500 errors; callers
// that want to treat an opaque 500 as overflow must check separately with
// additional context (e.g., session token counts).
func IsContextOverflowError(err error) bool {
	if err == nil {
		return false
	}

	// Already wrapped
	if _, ok := errors.AsType[*ContextOverflowError](err); ok {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	for _, pattern := range contextOverflowPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// statusCodeRegex matches HTTP status codes in error messages (e.g., "429", "500", ": 429 ")
var statusCodeRegex = regexp.MustCompile(`\b([45]\d{2})\b`)

// ExtractHTTPStatusCode attempts to extract an HTTP status code from the error
// using regex parsing of the error message. This is a fallback for providers
// whose errors are not yet wrapped in *StatusError (the preferred path).
//
// The regex matches 4xx/5xx codes at word boundaries
// (e.g., "429 Too Many Requests", "500 Internal Server Error").
// Returns 0 if no status code found.
func ExtractHTTPStatusCode(err error) int {
	if err == nil {
		return 0
	}

	// Check for *StatusError first (preferred structured path).
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode
	}

	// Fallback: extract from error message using regex.
	// OpenAI SDK error format: `POST "/v1/...": 429 Too Many Requests {...}`
	matches := statusCodeRegex.FindStringSubmatch(err.Error())
	if len(matches) >= 2 {
		if code, err := strconv.Atoi(matches[1]); err == nil {
			return code
		}
	}

	return 0
}

// IsRetryableStatusCode determines if an HTTP status code is retryable.
// Retryable means we should retry the SAME model with exponential backoff.
//
// Retryable status codes:
// - 5xx (server errors): 500, 502, 503, 504
// - 529 (Anthropic overloaded)
// - 408 (request timeout)
//
// Non-retryable status codes (skip to next model immediately):
// - 429 (rate limit) - provider is explicitly telling us to back off
// - 4xx client errors (400, 401, 403, 404) - won't get better with retry
func IsRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case 500, 502, 503, 504: // Server errors
		return true
	case 529: // Anthropic overloaded
		return true
	case 408: // Request timeout
		return true
	case 429: // Rate limit - NOT retryable, skip to next model
		return false
	default:
		return false
	}
}

// isRetryableModelError determines if an error should trigger a retry of the SAME model.
// It is used as a fallback by ClassifyModelError when no *StatusError is present.
//
// Retryable errors (retry same model with backoff):
// - Network timeouts
// - Temporary network errors
// - HTTP 5xx errors (server errors)
// - HTTP 529 (Anthropic overloaded)
// - HTTP 408 (request timeout)
//
// Non-retryable errors (skip to next model in chain immediately):
// - Context cancellation
// - HTTP 429 (rate limit) - provider is explicitly rate limiting us
// - HTTP 4xx errors (client errors)
// - Authentication errors
// - Invalid request errors
//
// The key distinction is: 429 means "you're calling too fast, slow down" which
// suggests we should try a different model, not keep hammering the same one.
func isRetryableModelError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation is never retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Context overflow errors are never retryable — the context hasn't changed
	// between attempts, so retrying the same oversized payload will always fail.
	// This avoids wasting time on 3 attempts + exponential backoff.
	if IsContextOverflowError(err) {
		slog.Debug("Context overflow error, not retryable", "error", err)
		return false
	}

	// First, try to extract HTTP status code from known SDK error types
	if statusCode := ExtractHTTPStatusCode(err); statusCode != 0 {
		retryable := IsRetryableStatusCode(statusCode)
		slog.Debug("Classified error by status code",
			"status_code", statusCode,
			"retryable", retryable)
		return retryable
	}

	// Check for network errors
	if netErr, ok := errors.AsType[net.Error](err); ok {
		// Timeout errors are retryable
		if netErr.Timeout() {
			slog.Debug("Network timeout error, retryable", "error", err)
			return true
		}
	}

	// Fall back to message-pattern matching for errors without structured status codes
	errMsg := strings.ToLower(err.Error())

	// Retryable patterns (timeout, network issues)
	// NOTE: Numeric status codes (500, 502, etc.) are already handled by
	// ExtractHTTPStatusCode + IsRetryableStatusCode above; they are not
	// duplicated here to avoid false positives on arbitrary numbers in
	// error messages (e.g., "processed 500 items").
	retryablePatterns := []string{
		"timeout",               // Generic timeout
		"connection reset",      // Connection reset
		"connection refused",    // Connection refused
		"no such host",          // DNS failure
		"temporary failure",     // Temporary failure
		"service unavailable",   // Service unavailable
		"internal server error", // Server error
		"bad gateway",           // Gateway error
		"gateway timeout",       // Gateway timeout
		"overloaded",            // Server overloaded
		"overloaded_error",      // Server overloaded
		"other side closed",     // Connection closed by peer
		"fetch failed",          // Network fetch failure
		"reset before headers",  // Connection reset before headers received
		"upstream connect",      // Upstream connection error
		"internal_error",        // HTTP/2 INTERNAL_ERROR (stream-level)
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			slog.Debug("Matched retryable error pattern", "pattern", pattern)
			return true
		}
	}

	// Non-retryable patterns (skip to next model immediately)
	// NOTE: Numeric status codes (429, 401, etc.) are already handled by
	// ExtractHTTPStatusCode above; they are not duplicated here.
	nonRetryablePatterns := []string{
		"rate limit",        // Rate limit message
		"too many requests", // Rate limit message
		"throttl",           // Throttling (rate limiting)
		"quota",             // Quota exceeded
		"capacity",          // Capacity issues (often rate-limit related)
		"invalid",           // Invalid request
		"unauthorized",      // Auth error
		"authentication",    // Auth error
		"api key",           // API key error
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errMsg, pattern) {
			slog.Debug("Matched non-retryable error pattern", "pattern", pattern)
			return false
		}
	}

	// Default: don't retry unknown errors to be safe
	slog.Debug("Unknown error type, not retrying", "error", err)
	return false
}

// ParseRetryAfterHeader parses a Retry-After header value.
// Supports both seconds (integer) and HTTP-date formats per RFC 7231 §7.1.3.
// Returns 0 if the value is empty, invalid, or results in a non-positive duration.
func ParseRetryAfterHeader(value string) time.Duration {
	if value == "" {
		return 0
	}
	// Try integer seconds first (most common for rate limits)
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	// Try HTTP-date format
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// ClassifyModelError classifies an error for the retry/fallback decision.
//
// If the error chain contains a *StatusError (wrapped by provider adapters),
// its StatusCode and RetryAfter fields are used directly — no provider-specific
// imports needed in the caller.
//
// Returns:
//   - retryable=true:    retry the SAME model with backoff (5xx, timeouts)
//   - rateLimited=true:  it's a 429 error; caller decides retry vs fallback based on config
//   - retryAfter:        Retry-After duration from the provider (only set for 429)
//
// When rateLimited=true, retryable is always false — the caller is responsible for
// deciding whether to retry (when no fallback is configured) or skip to the next
// model (when fallbacks are available).
func ClassifyModelError(err error) (retryable, rateLimited bool, retryAfter time.Duration) {
	if err == nil {
		return false, false, 0
	}

	// Context cancellation and deadline are never retryable.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false, false, 0
	}

	// Context overflow errors are never retryable — retrying the same oversized
	// payload will always fail.
	if IsContextOverflowError(err) {
		return false, false, 0
	}

	// Primary path: typed StatusError wrapped by provider adapters.
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		if statusErr.StatusCode == http.StatusTooManyRequests {
			return false, true, statusErr.RetryAfter
		}
		return IsRetryableStatusCode(statusErr.StatusCode), false, 0
	}

	// Fallback: providers that don't yet wrap (e.g. Bedrock), or non-provider
	// errors (network, pattern matching).
	statusCode := ExtractHTTPStatusCode(err)
	if statusCode == http.StatusTooManyRequests {
		return false, true, 0 // No Retry-After without StatusError
	}
	if statusCode != 0 {
		return IsRetryableStatusCode(statusCode), false, 0
	}
	return isRetryableModelError(err), false, 0
}

// CalculateBackoff returns the backoff duration for a given attempt (0-indexed).
// Uses exponential backoff with jitter.
func CalculateBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Calculate exponential delay
	delay := float64(backoffBaseDelay)
	for range attempt {
		delay *= backoffFactor
	}

	// Cap at max delay
	if delay > float64(backoffMaxDelay) {
		delay = float64(backoffMaxDelay)
	}

	// Add jitter (±10%)
	jitter := delay * backoffJitter * (2*rand.Float64() - 1)
	delay += jitter

	return time.Duration(delay)
}

// SleepWithContext sleeps for the specified duration, returning early if context is cancelled.
// Returns true if the sleep completed, false if it was interrupted by context cancellation.
func SleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// FormatError returns a user-friendly error message for model errors.
// Context overflow gets a dedicated actionable message; all other errors
// pass through their original message.
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	// Context overflow gets a dedicated, actionable message.
	if _, ok := errors.AsType[*ContextOverflowError](err); ok {
		return "The conversation has exceeded the model's context window and automatic compaction is not enabled. " +
			"Try running /compact to reduce the conversation size, or start a new session."
	}

	return err.Error()
}
