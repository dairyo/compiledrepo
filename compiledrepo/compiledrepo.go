package compiledrepo

import (
	"context"
	"iter"
)

// Loader is an interface for loading raw data associated with an ID.
type Loader interface {
	// Load retrieves the raw data for the given id.
	Load(ctx context.Context, id string) ([]byte, error)
}

// Compiler is a function type that compiles raw bytes into a typed value T.
type Compiler[T any] func([]byte) (T, error)

// Preloader is an interface for discovering all IDs that should be preloaded.
type Preloader interface {
	// All returns an iterator over all IDs to be preloaded and any associated error.
	All(ctx context.Context) iter.Seq2[string, error]
}
