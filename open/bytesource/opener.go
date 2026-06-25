package bytesource

import (
	"bytes"
	"context"
	"fmt"

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

// Open retrieves the data associated with the key and returns it as a bytes.Reader.
// It returns an error wrapped with compiledrepo.ErrOpen if the key is not found.
func (o *Opener[K]) Open(ctx context.Context, key K) (*bytes.Reader, error) {
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

	return bytes.NewReader(data), nil
}
