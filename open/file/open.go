package file

import (
	"context"
	"fmt"
	"os"

	"github.com/dairyo/compiledrepo"
)

// Open implements compiledrepo.Opener for local files.
type Open struct{}

// NewOpen creates a new Open instance.
func NewOpen() *Open {
	return &Open{}
}

// Open opens the file at the given path.
// It returns the file handle or an error wrapped with compiledrepo.ErrOpen.
func (o *Open) Open(ctx context.Context, path string) (*os.File, error) {
	// Check for context cancellation before starting the operation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", compiledrepo.ErrOpen, err)
	}

	return f, nil
}
