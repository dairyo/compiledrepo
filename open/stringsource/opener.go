package stringsource

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/dairyo/compiledrepo"
)

// Opener implements a simple resource opener that retrieves strings from an internal map.
type Opener[K comparable] struct {
	data map[K]string
}

// NewOpener creates a new instance of Opener with the provided data.
func NewOpener[K comparable](data map[K]string) *Opener[K] {
	return &Opener[K]{
		data: data,
	}
}

// Open retrieves the resource associated with the given key.
// It returns an io.ReadCloser for the resource or an error if the key is not found.
// If the context is cancelled, it returns the context error.
func (o *Opener[K]) Open(ctx context.Context, key K) (io.ReadCloser, error) {
	// Check for context cancellation before processing.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	val, ok := o.data[key]
	if !ok {
		return nil, fmt.Errorf("%w: key not found", compiledrepo.ErrOpen)
	}

	// Wrap strings.Reader with io.NopCloser to satisfy io.ReadCloser.
	return io.NopCloser(strings.NewReader(val)), nil
}
