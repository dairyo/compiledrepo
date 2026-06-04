package compiledrepo

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
			want error
		}{
			{"ErrInvalidArg is detectable", ErrInvalidArg, ErrInvalidArg},
			{"ErrNotFound is detectable", ErrNotFound, ErrNotFound},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if !errors.Is(tt.err, tt.want) {
					t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, tt.want)
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
			want error
		}{
			{"ErrInvalidArg is not ErrNotFound", ErrInvalidArg, ErrNotFound},
			{"ErrNotFound is not ErrInvalidArg", ErrNotFound, ErrInvalidArg},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if errors.Is(tt.err, tt.want) {
					t.Errorf("errors.Is(%v, %v) = true, want false", tt.err, tt.want)
				}
			})
		}
	})
}
