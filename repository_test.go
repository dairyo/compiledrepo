package compiledrepo

import (
	"context"
	"io"
	"testing"
)

// RepoMockOpener is a simple mock for testing Repository initialization.
type RepoMockOpener[K comparable, R io.ReadCloser] struct{}

func (m *RepoMockOpener[K, R]) Open(ctx context.Context, key K) (R, error) {
	var zero R
	return zero, nil
}

// RepoMockCompiler is a simple mock for testing Repository initialization.
type RepoMockCompiler[R io.ReadCloser, V any] struct{}

func (m *RepoMockCompiler[R, V]) Compile(ctx context.Context, r R) (V, error) {
	var zero V
	return zero, nil
}

func TestNewRepository(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name string
		}{
			{
				name: "InitializeWithMocks",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				opener := &RepoMockOpener[string, io.ReadCloser]{}
				compiler := &RepoMockCompiler[io.ReadCloser, string]{}

				repo := NewRepository(opener, compiler)

				if repo == nil {
					t.Fatal("expected repository instance, got nil")
				}
				if repo.opener != opener {
					t.Errorf("expected opener %v, got %v", opener, repo.opener)
				}
				if repo.compiler != compiler {
					t.Errorf("expected compiler %v, got %v", compiler, repo.compiler)
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		// NewRepository does not currently return an error.
	})
}
