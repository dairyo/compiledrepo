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
			want string
		}{
			{"ErrInvalidArg", ErrInvalidArg, "invalid argument"},
			{"ErrNotFound", ErrNotFound, "resource not found"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.err == nil {
					t.Fatal("expected error to be non-nil")
				}
				if tt.err.Error() != tt.want {
					t.Errorf("expected error message %q, got %q", tt.want, tt.err.Error())
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		// Sentinel errors themselves don't have "error cases" in the sense of failing logic,
		// but we can verify they are distinct.
		if errors.Is(ErrInvalidArg, ErrNotFound) {
			t.Error("ErrInvalidArg and ErrNotFound should be distinct")
		}
	})
}
