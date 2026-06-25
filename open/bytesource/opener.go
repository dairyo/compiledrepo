package bytesource

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/dairyo/compiledrepo"
)

// Opener implements compiledrepo.Opener for byte sources.
type Opener[K comparable] struct {
	store map[K][]byte
}

// NewOpener creates a new Opener instance with the provided store.
func NewOpener[K comparable](store map[K][]byte) *Opener[K] {
	return &Opener[K]{
		store: store,
	}
}

// Open retrieves the data associated with the key and returns it as an io.ReadCloser.
// It returns an error wrapped with compiledrepo.ErrOpen if the key is not found.
func (o *Opener[K]) Open(ctx context.Context, key K) (io.ReadCloser, error) {
	// Check for context cancellation before starting the operation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, ok := o.store[key]
	if !ok {
		return nil, fmt.Errorf("%w: key not found", compiledrepo.ErrOpen)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}
