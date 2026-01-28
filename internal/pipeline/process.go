// Package pipeline implements event processing stages: decode -> validate -> process.
package pipeline

import (
	"context"

	"go.uber.org/zap"
)

// Process executes the (mock) business logic for the decoded and validated payload.
// For Stage 0.5 and later this is deterministic and logs the payload fields.
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

	logger.Info("Processed event payload",
		zap.String("id", payload.Id),
	)
	return nil
}
