// Package pipeline implements event processing stages: decode -> validate -> process.
package pipeline

import (
	"context"
	"encoding/json"
	"errors"
)

// Payload is a minimal decoded event payload used for demonstration.
// It matches the expected JSON structure: {"Id":"..."}.
type Payload struct {
	Id string `json:"Id"`
}

// Decode unmarshals a JSON payload from the provided message value.
// It returns a typed *DecodeError for malformed JSON or invalid input.
func Decode(ctx context.Context, value []byte) (*Payload, error) {
	if err := ctx.Err(); err != nil {
		return nil, ErrContextCanceled
	}

	var p Payload
	if err := json.Unmarshal(value, &p); err != nil {
		// Preserve underlying error for errors.Is/As checks.
		return nil, &DecodeError{Err: err}
	}

	// We again check for context cancellation to report if decoding canceled during the work.
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, ErrContextCanceled
	}

	return &p, nil
}
