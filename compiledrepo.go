package compiledrepo

import (
	"context"
	"io"
	"iter"
)

// Opener defines the interface for opening a resource identified by key K,
// returning it as a ReadCloser R.
type Opener[K comparable, R io.ReadCloser] interface {
	// Open opens the resource identified by key.
	Open(ctx context.Context, key K) (R, error)
}

// Compiler defines the interface for compiling/transforming a resource R
// into a target value V.
type Compiler[R io.Reader, V any] interface {
	// Compile transforms the provided resource into the target type.
	Compile(ctx context.Context, r R) (V, error)
}

// KeyIterator defines the interface for iterating over all keys of type K.
type KeyIterator[K comparable] interface {
	// All returns a sequence of keys and potential errors.
	All(ctx context.Context) iter.Seq2[K, error]
}
