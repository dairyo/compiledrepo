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
			{"ErrInvalidArg", ErrInvalidArg, ErrInvalidArg},
			{"ErrNotFound", ErrNotFound, ErrNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if !errors.Is(tt.err, tt.want) {
					t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, tt.want)
				}
			})
		}
	})
}
