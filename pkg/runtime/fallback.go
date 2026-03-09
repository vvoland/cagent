package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker-agent/pkg/agent"
	"github.com/docker/docker-agent/pkg/chat"
	"github.com/docker/docker-agent/pkg/model/provider"
	"github.com/docker/docker-agent/pkg/model/provider/options"
	"github.com/docker/docker-agent/pkg/modelerrors"
	"github.com/docker/docker-agent/pkg/modelsdev"
	"github.com/docker/docker-agent/pkg/session"
	"github.com/docker/docker-agent/pkg/tools"
)

// fallbackCooldownState tracks when we should stick with a fallback model
// instead of retrying the primary after a non-retryable error (e.g., 429).
type fallbackCooldownState struct {
	// fallbackIndex is the index in the fallback chain to start from (0 = first fallback, -1 = primary)
	fallbackIndex int
	// until is when the cooldown expires and we should retry the primary
	until time.Time
}

// modelWithFallback holds a provider and its identification for logging
type modelWithFallback struct {
	provider   provider.Provider
	isFallback bool
	index      int // index in fallback list (-1 for primary)
}

// buildModelChain returns the ordered list of models to try: primary first, then fallbacks.
func buildModelChain(primary provider.Provider, fallbacks []provider.Provider) []modelWithFallback {
	chain := make([]modelWithFallback, 0, 1+len(fallbacks))
	chain = append(chain, modelWithFallback{
		provider:   primary,
		isFallback: false,
		index:      -1,
	})
	for i, fb := range fallbacks {
		chain = append(chain, modelWithFallback{
			provider:   fb,
			isFallback: true,
			index:      i,
		})
	}
	return chain
}

// logFallbackAttempt logs information about a fallback attempt
func logFallbackAttempt(agentName string, model modelWithFallback, attempt, maxRetries int, err error) {
	if model.isFallback {
		slog.Warn("Fallback model attempt",
			"agent", agentName,
			"model", model.provider.ID(),
			"fallback_index", model.index,
			"attempt", attempt+1,
			"max_retries", maxRetries+1,
			"previous_error", err)
	} else {
		slog.Warn("Primary model failed, trying fallbacks",
			"agent", agentName,
			"model", model.provider.ID(),
			"error", err)
	}
}

// logRetryBackoff logs when we're backing off before a retry
func logRetryBackoff(agentName, modelID string, attempt int, backoff time.Duration) {
	slog.Debug("Backing off before retry",
		"agent", agentName,
		"model", modelID,
		"attempt", attempt+1,
		"backoff", backoff)
}

// getCooldownState returns the current cooldown state for an agent (thread-safe).
// Returns nil if no cooldown is active or if cooldown has expired.
// Expired entries are evicted to prevent stale state accumulation.
func (r *LocalRuntime) getCooldownState(agentName string) *fallbackCooldownState {
	r.fallbackCooldownsMux.Lock()
	defer r.fallbackCooldownsMux.Unlock()

	state := r.fallbackCooldowns[agentName]
	if state == nil {
		return nil
	}

	// Check if cooldown has expired; evict if so
	if time.Now().After(state.until) {
		delete(r.fallbackCooldowns, agentName)
		return nil
	}

	return state
}

// setCooldownState sets the cooldown state for an agent (thread-safe).
func (r *LocalRuntime) setCooldownState(agentName string, fallbackIndex int, cooldownDuration time.Duration) {
	r.fallbackCooldownsMux.Lock()
	defer r.fallbackCooldownsMux.Unlock()

	r.fallbackCooldowns[agentName] = &fallbackCooldownState{
		fallbackIndex: fallbackIndex,
		until:         time.Now().Add(cooldownDuration),
	}

	slog.Info("Fallback cooldown activated",
		"agent", agentName,
		"fallback_index", fallbackIndex,
		"cooldown", cooldownDuration,
		"until", r.fallbackCooldowns[agentName].until.Format(time.RFC3339))
}

// clearCooldownState clears the cooldown state for an agent (thread-safe).
func (r *LocalRuntime) clearCooldownState(agentName string) {
	r.fallbackCooldownsMux.Lock()
	defer r.fallbackCooldownsMux.Unlock()

	if _, exists := r.fallbackCooldowns[agentName]; exists {
		delete(r.fallbackCooldowns, agentName)
		slog.Debug("Fallback cooldown cleared", "agent", agentName)
	}
}

// getEffectiveCooldown returns the cooldown duration to use for an agent.
// Uses the agent's configured cooldown, or the default if not set.
func getEffectiveCooldown(a *agent.Agent) time.Duration {
	cooldown := a.FallbackCooldown()
	if cooldown == 0 {
		return modelerrors.DefaultCooldown
	}
	return cooldown
}

// getEffectiveRetries returns the number of retries to use for the agent.
// If no retries are explicitly configured (retries == 0), returns
// the default to provide sensible retry behavior out of the box.
// This ensures that transient errors (e.g., Anthropic 529 overloaded) are
// retried even when no fallback models are configured.
//
// Note: Users who explicitly want 0 retries can set retries: -1 in their config
// (though this is an edge case - most users want some retries for resilience).
func getEffectiveRetries(a *agent.Agent) int {
	retries := a.FallbackRetries()
	// -1 means "explicitly no retries" (workaround for Go's zero value)
	if retries < 0 {
		return 0
	}
	// 0 means "use default" - always provide retries for transient error resilience
	if retries == 0 {
		return modelerrors.DefaultRetries
	}
	return retries
}

// tryModelWithFallback attempts to create a stream and get a response using the primary model,
// falling back to configured fallback models if the primary fails.
//
// Retry behavior:
// - Retryable errors (5xx, timeouts): retry the same model with exponential backoff
// - Non-retryable errors (429, 4xx): skip to the next model in the chain immediately
//
// Cooldown behavior:
//   - When the primary fails with a non-retryable error and a fallback succeeds, the runtime
//     "sticks" with that fallback for a configurable cooldown period.
//   - During cooldown, subsequent calls skip the primary and start from the pinned fallback.
//   - When cooldown expires, the primary is tried again; if it succeeds, cooldown is cleared.
//
// Returns the stream result, the model that was used, and any error.
func (r *LocalRuntime) tryModelWithFallback(
	ctx context.Context,
	a *agent.Agent,
	primaryModel provider.Provider,
	messages []chat.Message,
	agentTools []tools.Tool,
	sess *session.Session,
	m *modelsdev.Model,
	events chan Event,
) (streamResult, provider.Provider, error) {
	// Clone fallback models with the same thinking override as the primary model.
	// The primary model was already cloned with options.WithThinking(sess.Thinking)
	// in the main runtime loop, so we apply the same to fallbacks for consistency.
	rawFallbacks := a.FallbackModels()
	fallbackModels := make([]provider.Provider, len(rawFallbacks))
	for i, fb := range rawFallbacks {
		fallbackModels[i] = provider.CloneWithOptions(ctx, fb, options.WithThinking(sess.Thinking))
	}

	fallbackRetries := getEffectiveRetries(a)

	// Build the chain of models to try: primary (index 0) + fallbacks (index 1+)
	modelChain := buildModelChain(primaryModel, fallbackModels)

	// Check if we're in a cooldown period and should skip the primary
	startIndex := 0
	inCooldown := false
	cooldownState := r.getCooldownState(a.Name())
	if cooldownState != nil && len(fallbackModels) > cooldownState.fallbackIndex {
		// We're in cooldown - start from the pinned fallback (skip primary)
		startIndex = cooldownState.fallbackIndex + 1 // +1 because index 0 is primary
		inCooldown = true
		slog.Debug("Skipping primary due to cooldown",
			"agent", a.Name(),
			"start_from_fallback_index", cooldownState.fallbackIndex,
			"cooldown_until", cooldownState.until.Format(time.RFC3339))
	}

	var lastErr error
	primaryFailedWithNonRetryable := false

	for chainIdx := startIndex; chainIdx < len(modelChain); chainIdx++ {
		modelEntry := modelChain[chainIdx]

		// Each model in the chain gets (1 + retries) attempts for retryable errors.
		// Non-retryable errors (429, 4xx) skip immediately to the next model.
		maxAttempts := 1 + fallbackRetries

		for attempt := range maxAttempts {
			// Check context before each attempt
			if ctx.Err() != nil {
				return streamResult{}, nil, ctx.Err()
			}

			// Apply backoff before retry (not on first attempt of each model)
			if attempt > 0 {
				backoff := modelerrors.CalculateBackoff(attempt - 1)
				logRetryBackoff(a.Name(), modelEntry.provider.ID(), attempt, backoff)
				if !modelerrors.SleepWithContext(ctx, backoff) {
					return streamResult{}, nil, ctx.Err()
				}
			}

			// Emit fallback event when transitioning to a new model (but not when starting in cooldown)
			if chainIdx > startIndex && attempt == 0 {
				logFallbackAttempt(a.Name(), modelEntry, attempt, fallbackRetries, lastErr)
				// Get the previous model's ID for the event
				prevModelID := modelChain[chainIdx-1].provider.ID()
				reason := ""
				if lastErr != nil {
					reason = lastErr.Error()
				}
				events <- ModelFallback(
					a.Name(),
					prevModelID,
					modelEntry.provider.ID(),
					reason,
					attempt+1,
					maxAttempts,
				)
			}

			slog.Debug("Creating chat completion stream",
				"agent", a.Name(),
				"model", modelEntry.provider.ID(),
				"is_fallback", modelEntry.isFallback,
				"in_cooldown", inCooldown,
				"attempt", attempt+1)

			stream, err := modelEntry.provider.CreateChatCompletionStream(ctx, messages, agentTools)
			if err != nil {
				lastErr = err

				// Context cancellation is never retryable
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return streamResult{}, nil, err
				}

				// Check if error is retryable
				if !modelerrors.IsRetryableModelError(err) {
					slog.Error("Non-retryable error creating stream",
						"agent", a.Name(),
						"model", modelEntry.provider.ID(),
						"error", err)

					// Track if primary failed with non-retryable error
					if !modelEntry.isFallback {
						primaryFailedWithNonRetryable = true
					}

					// Skip to next model in chain
					break
				}

				slog.Warn("Retryable error creating stream",
					"agent", a.Name(),
					"model", modelEntry.provider.ID(),
					"attempt", attempt+1,
					"max_attempts", maxAttempts,
					"error", err)
				continue
			}

			// Stream created successfully, now handle it
			slog.Debug("Processing stream", "agent", a.Name(), "model", modelEntry.provider.ID())
			res, err := r.handleStream(ctx, stream, a, agentTools, sess, m, events)
			if err != nil {
				lastErr = err

				// Context cancellation stops everything
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return streamResult{}, nil, err
				}

				// Check if stream error is retryable
				if !modelerrors.IsRetryableModelError(err) {
					slog.Error("Non-retryable error handling stream",
						"agent", a.Name(),
						"model", modelEntry.provider.ID(),
						"error", err)

					// Track if primary failed with non-retryable error
					if !modelEntry.isFallback {
						primaryFailedWithNonRetryable = true
					}

					break
				}

				slog.Warn("Retryable error handling stream",
					"agent", a.Name(),
					"model", modelEntry.provider.ID(),
					"attempt", attempt+1,
					"max_attempts", maxAttempts,
					"error", err)
				continue
			}

			// Success!
			// Handle cooldown state based on which model succeeded
			switch {
			case modelEntry.isFallback && primaryFailedWithNonRetryable:
				// Primary failed with non-retryable error, fallback succeeded.
				// Set cooldown to stick with this fallback.
				r.setCooldownState(a.Name(), modelEntry.index, getEffectiveCooldown(a))
			case !modelEntry.isFallback:
				// Primary succeeded - clear any existing cooldown.
				// This handles both normal success and recovery after cooldown expires.
				r.clearCooldownState(a.Name())
			}

			return res, modelEntry.provider, nil
		}
	}

	// All models and retries exhausted.
	// If the last error (or any error in the chain) was a context overflow,
	// wrap it in a ContextOverflowError so the caller can auto-compact.
	if lastErr != nil {
		wrapped := fmt.Errorf("all models failed: %w", lastErr)
		if modelerrors.IsContextOverflowError(lastErr) {
			return streamResult{}, nil, &modelerrors.ContextOverflowError{Underlying: wrapped}
		}
		return streamResult{}, nil, wrapped
	}
	return streamResult{}, nil, errors.New("all models failed with unknown error")
}
