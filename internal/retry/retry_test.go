// Package retry provides retry logic with exponential backoff for event processing
package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/config"
)

func TestDoWithRetry_Success(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelayMs: 10 * time.Millisecond,
		MaxDelayMs:  100 * time.Millisecond,
		Multiplier:  2.0,
	}

	callCount := 0
	fn := func() error {
		callCount++
		return nil // Success on first attempt
	}

	err := DoWithRetry(context.Background(), &cfg, fn)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got: %d", callCount)
	}
}

func TestDoWithRetry_SuccessAfterRetries(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelayMs: 10 * time.Millisecond,
		MaxDelayMs:  100 * time.Millisecond,
		Multiplier:  2.0,
	}

	callCount := 0
	failUntil := 2
	fn := func() error {
		callCount++
		if callCount <= failUntil {
			return errors.New("transient error")
		}
		return nil // Success after 2 failures
	}

	err := DoWithRetry(context.Background(), &cfg, fn)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got: %d", callCount)
	}
}

func TestDoWithRetry_ExhaustedRetries(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 2,
		BaseDelayMs: 10 * time.Millisecond,
		MaxDelayMs:  100 * time.Millisecond,
		Multiplier:  2.0,
	}

	callCount := 0
	expectedErr := errors.New("permanent error")
	fn := func() error {
		callCount++
		return expectedErr
	}

	err := DoWithRetry(context.Background(), &cfg, fn)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("Expected ErrMaxRetriesExceeded, got: %v", err)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected wrapped original error, got: %v", err)
	}
	expectedCalls := cfg.MaxAttempts + 1 // initial + retries
	if callCount != expectedCalls {
		t.Errorf("Expected %d calls, got: %d", expectedCalls, callCount)
	}
}

func TestDoWithRetry_ContextCancelled(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 5,
		BaseDelayMs: 50 * time.Millisecond,
		MaxDelayMs:  500 * time.Millisecond,
		Multiplier:  2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("error")
	}

	err := DoWithRetry(ctx, &cfg, fn)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
	// Should not have called fn at all since context was already cancelled
	if callCount > 0 {
		t.Errorf("Expected 0 calls when context is pre-cancelled, got: %d", callCount)
	}
}

func TestDoWithRetry_ContextCancelledDuringRetry(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 5,
		BaseDelayMs: 100 * time.Millisecond,
		MaxDelayMs:  500 * time.Millisecond,
		Multiplier:  2.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("error")
	}

	err := DoWithRetry(ctx, &cfg, fn)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
	// Should have called fn once or twice before context timeout
	if callCount < 1 {
		t.Errorf("Expected at least 1 call, got: %d", callCount)
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.RetryConfig
		attempt  int
		expected time.Duration
	}{
		{
			name: "first retry",
			cfg: config.RetryConfig{
				BaseDelayMs: 100 * time.Millisecond,
				MaxDelayMs:  10000 * time.Millisecond,
				Multiplier:  2.0,
			},
			attempt:  0,
			expected: 100 * time.Millisecond,
		},
		{
			name: "second retry",
			cfg: config.RetryConfig{
				BaseDelayMs: 100 * time.Millisecond,
				MaxDelayMs:  10000 * time.Millisecond,
				Multiplier:  2.0,
			},
			attempt:  1,
			expected: 200 * time.Millisecond,
		},
		{
			name: "third retry",
			cfg: config.RetryConfig{
				BaseDelayMs: 100 * time.Millisecond,
				MaxDelayMs:  10000 * time.Millisecond,
				Multiplier:  2.0,
			},
			attempt:  2,
			expected: 400 * time.Millisecond,
		},
		{
			name: "capped at max delay",
			cfg: config.RetryConfig{
				BaseDelayMs: 100 * time.Millisecond,
				MaxDelayMs:  500 * time.Millisecond,
				Multiplier:  2.0,
			},
			attempt:  5, // would be 3200ms without cap
			expected: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := calculateBackoff(&tt.cfg, tt.attempt)
			if actual != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestDoWithRetry_NoRetries(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 0, // No retries
		BaseDelayMs: 10 * time.Millisecond,
		MaxDelayMs:  100 * time.Millisecond,
		Multiplier:  2.0,
	}

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("error")
	}

	err := DoWithRetry(context.Background(), &cfg, fn)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("Expected ErrMaxRetriesExceeded, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call (no retries), got: %d", callCount)
	}
}
