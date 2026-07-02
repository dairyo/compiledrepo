package file

import (
	"context"
	"fmt"
	"os"

	"github.com/dairyo/compiledrepo"
)

// Opener implements compiledrepo.Opener for local files.
type Opener struct{}

// NewOpener creates a new Opener instance.
func NewOpener() *Opener {
	return &Opener{}
}

// Open opens the file at the given path.
// It returns the file handle or an error wrapped with compiledrepo.ErrOpen.
func (o *Opener) Open(ctx context.Context, path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", compiledrepo.ErrOpen, err)
	}

	return f, nil
}
