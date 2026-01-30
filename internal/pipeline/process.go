// Package pipeline implements event processing stages: decode -> validate -> process.
package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"go.uber.org/zap"
)

// Process executes the (mock) business logic for the decoded and validated payload.
func Process(ctx context.Context, payload *Payload, logger *zap.Logger) error {
	if err := ctx.Err(); err != nil {
		return ErrContextCanceled
	}
	if payload == nil {
		return &ProcessError{Err: &ValidationError{Field: "Payload", Reason: "is nil"}}
	}
	if logger == nil {
		return &ProcessError{Err: &ValidationError{Field: "Logger", Reason: "is nil"}}
	}

	// Simulate processing: derive a hash from payload (CPU-bound)
	// It is implemented to simulate a CPU-bound operation.
	for range 1000 {
		if err := ctx.Err(); err != nil {
			return ErrContextCanceled
		}
		h := sha256.Sum256([]byte(payload.Id))
		_ = hex.EncodeToString(h[:]) // prevent optimization
	}

	logger.Info("Processed event payload", zap.String("id", payload.Id))
	return nil
}
