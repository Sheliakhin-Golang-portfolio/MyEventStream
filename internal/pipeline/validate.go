// Package pipeline implements event processing stages: decode -> validate -> process.
package pipeline

import (
	"context"
	"strings"
)

// Validate checks required fields on the decoded payload.
// It returns a typed *ValidationError for validation failures.
func Validate(ctx context.Context, payload *Payload) error {
	if err := ctx.Err(); err != nil {
		return ErrContextCanceled
	}
	if payload == nil {
		return &ValidationError{Field: "Payload", Reason: "is nil"}
	}

	if strings.TrimSpace(payload.Id) == "" {
		return &ValidationError{Field: "Id", Reason: "is required"}
	}

	return nil
}

