package pipeline

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestProcess(t *testing.T) {
	t.Parallel()

	newObservedLogger := func() (*zap.Logger, *observer.ObservedLogs) {
		core, logs := observer.New(zapcore.InfoLevel)
		return zap.New(core), logs
	}

	tests := []struct {
		name         string
		ctx          context.Context
		payload      *Payload
		logger       func() *zap.Logger
		wantIsErr    bool
		wantAsProc   bool
		wantErr      error
		wantLogCount int
		wantLogID    string
	}{
		{
			name:         "success_logs_event",
			ctx:          context.Background(),
			payload:      &Payload{Id: "abc"},
			logger:       func() *zap.Logger { l, _ := newObservedLogger(); return l },
			wantIsErr:    false,
			wantLogCount: 1,
			wantLogID:    "abc",
		},
		{
			name:      "nil_payload",
			ctx:       context.Background(),
			payload:   nil,
			logger:    func() *zap.Logger { l, _ := newObservedLogger(); return l },
			wantIsErr: true,
			wantAsProc: true,
		},
		{
			name:      "nil_logger",
			ctx:       context.Background(),
			payload:   &Payload{Id: "abc"},
			logger:    func() *zap.Logger { return nil },
			wantIsErr: true,
			wantAsProc: true,
		},
		{
			name:      "context_canceled",
			ctx:       func() context.Context { c, cancel := context.WithCancel(context.Background()); cancel(); return c }(),
			payload:   &Payload{Id: "abc"},
			logger:    func() *zap.Logger { l, _ := newObservedLogger(); return l },
			wantIsErr: true,
			wantErr:   ErrContextCanceled,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs *observer.ObservedLogs
			var logger *zap.Logger
			if tt.logger != nil {
				// If using observed logger, keep logs handle.
				if tt.name == "success_logs_event" {
					logger, logs = newObservedLogger()
				} else {
					logger = tt.logger()
				}
			}

			err := Process(tt.ctx, tt.payload, logger)
			if tt.wantIsErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected errors.Is(err, %v)=true; got err=%v", tt.wantErr, err)
				}
				if tt.wantAsProc {
					var pe *ProcessError
					if !errors.As(err, &pe) {
						t.Fatalf("expected error to be *ProcessError, got %T (%v)", err, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if logs == nil {
				t.Fatalf("expected observed logs handle, got nil")
			}
			if logs.Len() != tt.wantLogCount {
				t.Fatalf("expected %d logs, got %d", tt.wantLogCount, logs.Len())
			}
			entries := logs.All()
			if len(entries) != tt.wantLogCount {
				t.Fatalf("expected %d log entries, got %d", tt.wantLogCount, len(entries))
			}
			if entries[0].Message != "Processed event payload" {
				t.Fatalf("expected log message %q, got %q", "Processed event payload", entries[0].Message)
			}
			ctxMap := entries[0].ContextMap()
			if gotID, _ := ctxMap["id"].(string); gotID != tt.wantLogID {
				t.Fatalf("expected log field id=%q, got %v", tt.wantLogID, ctxMap["id"])
			}
		})
	}
}

