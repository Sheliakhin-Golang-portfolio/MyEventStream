// Package pipeline implements event processing stages: decode -> validate -> process.
package pipeline

import "errors"

// Common pipeline errors.
var (
	// ErrContextCanceled indicates the caller's context was canceled.
	// Stages may return this error directly (or wrapped) when ctx.Done() is signaled.
	ErrContextCanceled = errors.New("context canceled")
)

// DecodeError represents a failure in the decode stage.
// It wraps the underlying decoder/unmarshal error.
type DecodeError struct {
	Err error
}

func (e *DecodeError) Error() string {
	if e == nil || e.Err == nil {
		return "decode failed"
	}
	return "decode failed: " + e.Err.Error()
}

// Unwrap returns the underlying error.
// This is useful for error chaining and propagation.
func (e *DecodeError) Unwrap() error { return e.Err }

// ValidationError represents a failure in the validate stage.
// Field is the name of the invalid field; Reason describes why.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	errMsg := "validation failed"
	if e != nil && e.Field != "" {
		errMsg += ": " + e.Field
	}
	if e != nil && e.Reason != "" {
		errMsg += ": " + e.Reason
	}
	return errMsg
}

// ProcessError represents a failure in the process stage.
// It wraps the underlying processing error.
type ProcessError struct {
	Err error
}

func (e *ProcessError) Error() string {
	if e == nil || e.Err == nil {
		return "process failed"
	}
	return "process failed: " + e.Err.Error()
}

func (e *ProcessError) Unwrap() error { return e.Err }
