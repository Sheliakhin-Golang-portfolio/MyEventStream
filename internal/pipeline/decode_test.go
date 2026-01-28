package pipeline

import (
	"context"
	"errors"
	"testing"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ctx       context.Context
		value     []byte
		wantID    string
		wantIsErr bool
		wantAsDec bool
		wantErr   error
	}{
		{
			name:      "valid_json",
			ctx:       context.Background(),
			value:     []byte(`{"Id":"abc"}`),
			wantID:    "abc",
			wantIsErr: false,
		},
		{
			name:      "missing_id_field_is_ok_in_decode",
			ctx:       context.Background(),
			value:     []byte(`{}`),
			wantID:    "",
			wantIsErr: false,
		},
		{
			name:      "invalid_json",
			ctx:       context.Background(),
			value:     []byte(`{"Id":`),
			wantIsErr: true,
			wantAsDec: true,
		},
		{
			name:      "empty_payload_is_decode_error",
			ctx:       context.Background(),
			value:     []byte(``),
			wantIsErr: true,
			wantAsDec: true,
		},
		{
			name:      "context_canceled",
			ctx:       func() context.Context { c, cancel := context.WithCancel(context.Background()); cancel(); return c }(),
			value:     []byte(`{"Id":"abc"}`),
			wantIsErr: true,
			wantErr:   ErrContextCanceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Decode(tt.ctx, tt.value)
			if tt.wantIsErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected errors.Is(err, %v)=true; got err=%v", tt.wantErr, err)
				}
				if tt.wantAsDec {
					var de *DecodeError
					if !errors.As(err, &de) {
						t.Fatalf("expected error to be *DecodeError, got %T (%v)", err, err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got == nil {
				t.Fatalf("expected payload, got nil")
			}
			if got.Id != tt.wantID {
				t.Fatalf("expected Id=%q, got %q", tt.wantID, got.Id)
			}
		})
	}
}
