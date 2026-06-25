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
			expected error
		}{
			{"ErrOpen", ErrOpen, ErrOpen},
			{"ErrCompile", ErrCompile, ErrCompile},
			{"ErrIterator", ErrIterator, ErrIterator},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if !errors.Is(tt.err, tt.expected) {
					t.Errorf("expected error %v, got %v", tt.expected, tt.err)
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name     string
			err      error
			expected error
		}{
			{"ErrOpenIsNotErrCompile", ErrOpen, ErrCompile},
			{"ErrOpenIsNotErrIterator", ErrOpen, ErrIterator},
			{"ErrCompileIsNotErrOpen", ErrCompile, ErrOpen},
			{"ErrCompileIsNotErrIterator", ErrCompile, ErrIterator},
			{"ErrIteratorIsNotErrOpen", ErrIterator, ErrOpen},
			{"ErrIteratorIsNotErrCompile", ErrIterator, ErrCompile},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if errors.Is(tt.err, tt.expected) {
					t.Errorf("expected error %v to not be %v", tt.err, tt.expected)
				}
			})
		}
	})
}
