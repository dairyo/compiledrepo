package compiledrepo

import (
	"testing"
)

func TestInterfaces(t *testing.T) {
	t.Run("OpenerAssignment", func(t *testing.T) {
		var _ Opener[string, *MockReadCloser] = &MockOpener[string, *MockReadCloser]{}
	})

	t.Run("CompilerAssignment", func(t *testing.T) {
		var _ Compiler[*MockReadCloser, string] = &MockCompiler[*MockReadCloser, string]{}
	})

	t.Run("IteratorAssignment", func(t *testing.T) {
		var _ KeyIterator[string] = &MockKeyIterator[string]{}
	})
}
