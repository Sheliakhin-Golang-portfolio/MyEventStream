// Package retry provides retry logic with exponential backoff for event processing
package retry

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/config"
)

// Errors
var (
	// ErrMaxRetriesExceeded is returned when all retry attempts have been exhausted
	ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded")
)

// DoWithRetry executes fn with retry logic according to the provided configuration.
// It returns ErrMaxRetriesExceeded wrapped with the last error if all retries fail.
func DoWithRetry(ctx context.Context, cfg *config.RetryConfig, fn func() error) error {
	var err error

	// Loop through the retry attempts
	for i := range cfg.MaxAttempts + 1 {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err = fn()
		if err == nil {
			return nil // Success
		}

		// If this was the last attempt, break the loop
		if i == cfg.MaxAttempts {
			break
		}

		// Calculate backoff delay for next retry
		delay := calculateBackoff(cfg, i)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next retry
		}
	}

	// Return the error if all retries failed
	return errors.Join(ErrMaxRetriesExceeded, err)
}

// calculateBackoff computes the backoff delay for a given attempt
func calculateBackoff(cfg *config.RetryConfig, attempt int) time.Duration {
	// Exponential backoff: baseDelayMs * (multiplier ^ attempt)
	delay := cfg.BaseDelayMs * time.Duration(math.Pow(cfg.Multiplier, float64(attempt)))

	// Cap at MaxDelay
	return min(delay, cfg.MaxDelayMs)
}
