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

// BasicOpener implements Opener[string, *MockReadCloser].
type BasicOpener struct{}

func (m *BasicOpener) Open(ctx context.Context, key string) (*MockReadCloser, error) {
	return &MockReadCloser{}, nil
}

// BasicCompiler implements Compiler[*MockReadCloser, string].
type BasicCompiler struct{}

func (m *BasicCompiler) Compile(ctx context.Context, r *MockReadCloser) (string, error) {
	return "compiled", nil
}

// BasicIterator implements KeyIterator[string].
type BasicIterator struct{}

func (m *BasicIterator) All(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		if !yield("key1", nil) {
			return
		}
	}
}

func TestInterfaces(t *testing.T) {
	t.Run("OpenerAssignment", func(t *testing.T) {
		var _ Opener[string, *MockReadCloser] = &BasicOpener{}
	})

	t.Run("CompilerAssignment", func(t *testing.T) {
		var _ Compiler[*MockReadCloser, string] = &BasicCompiler{}
	})

	t.Run("IteratorAssignment", func(t *testing.T) {
		var _ KeyIterator[string] = &BasicIterator{}
	})
}
