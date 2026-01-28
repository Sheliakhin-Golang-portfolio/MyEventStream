package pipeline

import (
	"context"
	"errors"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ctx       context.Context
		payload   *Payload
		wantIsErr bool
		wantAsVal bool
		wantErr   error
	}{
		{
			name:      "valid",
			ctx:       context.Background(),
			payload:   &Payload{Id: "abc"},
			wantIsErr: false,
		},
		{
			name:      "empty_id",
			ctx:       context.Background(),
			payload:   &Payload{Id: ""},
			wantIsErr: true,
			wantAsVal: true,
		},
		{
			name:      "whitespace_id",
			ctx:       context.Background(),
			payload:   &Payload{Id: "   "},
			wantIsErr: true,
			wantAsVal: true,
		},
		{
			name:      "nil_payload",
			ctx:       context.Background(),
			payload:   nil,
			wantIsErr: true,
			wantAsVal: true,
		},
		{
			name:      "context_canceled",
			ctx:       func() context.Context { c, cancel := context.WithCancel(context.Background()); cancel(); return c }(),
			payload:   &Payload{Id: "abc"},
			wantIsErr: true,
			wantErr:   ErrContextCanceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Validate(tt.ctx, tt.payload)
			if tt.wantIsErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected errors.Is(err, %v)=true; got err=%v", tt.wantErr, err)
				}
				if tt.wantAsVal {
					var ve *ValidationError
					if !errors.As(err, &ve) {
						t.Fatalf("expected error to be *ValidationError, got %T (%v)", err, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
