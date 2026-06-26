package compiledrepo

import (
	"context"
	"io"
	"iter"
	"testing"
)

// MockReadCloser is a simple mock for io.ReadCloser.
type MockReadCloser struct {
	io.ReadCloser
}

func (m *MockReadCloser) Close() error {
	return nil
}

// MockIterator implements KeyIterator[string].
type MockIterator struct{}

func (m *MockIterator) All(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		if !yield("key1", nil) {
			return
		}
	}
}

func TestInterfaces(t *testing.T) {
	t.Run("OpenerAssignment", func(t *testing.T) {
		var _ Opener[string, *MockReadCloser] = &MockOpener[string, *MockReadCloser]{}
	})

	t.Run("CompilerAssignment", func(t *testing.T) {
		var _ Compiler[*MockReadCloser, string] = &MockCompiler[*MockReadCloser, string]{}
	})

	t.Run("IteratorAssignment", func(t *testing.T) {
		var _ KeyIterator[string] = &MockIterator{}
	})
}
