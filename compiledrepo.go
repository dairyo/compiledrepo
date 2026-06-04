package compiledrepo

import (
	"context"
	"iter"
	"sync"

	"golang.org/x/sync/singleflight"
)

// Loader defines the interface for loading raw byte data from an external source.
type Loader interface {
	// Load retrieves the data associated with the given id.
	Load(ctx context.Context, id string) ([]byte, error)
}

// Compiler is a function type that converts raw byte data into a specific type T.
type Compiler[T any] func([]byte) (T, error)

// Preloader defines the interface for discovering all available resource IDs.
type Preloader interface {
	// All returns an iterator that yields all available resource IDs and potential errors.
	All(ctx context.Context) iter.Seq2[string, error]
}

// Repository manages the loading, compiling, and caching of resources of type T.
type Repository[T any] struct {
	loader   Loader
	compiler Compiler[T]
	cache    sync.Map
	sfg      singleflight.Group
}

// Registry provides a read-only snapshot of resources of type T.
type Registry[T any] struct {
	resources map[string]T
}
