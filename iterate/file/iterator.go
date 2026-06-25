package file

import (
	"context"
	"fmt"
	"iter"
	"os"

	"github.com/dairyo/compiledrepo"
)

// Iterator implements compiledrepo.KeyIterator[string] for a directory.
type Iterator struct {
	dirPath string
}

// NewIterator creates a new Iterator for the given directory path.
func NewIterator(dirPath string) *Iterator {
	return &Iterator{
		dirPath: dirPath,
	}
}

// All returns a sequence of filenames in the directory.
// It yields each file name as a key and nil as the error.
// If the directory cannot be read, it yields a wrapped compiledrepo.ErrIterator.
// It respects context cancellation.
func (it *Iterator) All(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		entries, err := os.ReadDir(it.dirPath)
		if err != nil {
			if !yield("", fmt.Errorf("%w: failed to read directory %s: %w", compiledrepo.ErrIterator, it.dirPath, err)) {
				return
			}
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			select {
			case <-ctx.Done():
				if !yield("", ctx.Err()) {
					return
				}
				return
			default:
			}

			if !yield(entry.Name(), nil) {
				return
			}
		}
	}
}
