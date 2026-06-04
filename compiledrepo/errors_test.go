package compiledrepo

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name     string
			err      error
			expected string
		}{
			{
				name:     "ErrInvalidArg",
				err:      ErrInvalidArg,
				expected: "invalid argument",
			},
			{
				name:     "ErrNotFound",
				err:      ErrNotFound,
				expected: "resource not found",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.err == nil {
					t.Fatal("sentinel error should not be nil")
				}
				if tt.err.Error() != tt.expected {
					t.Errorf("expected error message %q, got %q", tt.expected, tt.err.Error())
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		// Since sentinel errors are simple variables, there are no typical "ErrorCases"
		// for the definitions themselves, but we can verify they are distinct.
		if errors.Is(ErrInvalidArg, ErrNotFound) {
			t.Error("ErrInvalidArg should be distinct from ErrNotFound")
		}
	})
}
