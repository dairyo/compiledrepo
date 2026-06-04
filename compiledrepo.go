package compiledrepo

import (
	"context"
	"iter"
	"sync"

	"golang.org/x/sync/singleflight"
)

// Loader defines the interface for loading raw data from a source.
type Loader interface {
	Load(ctx context.Context, id string) ([]byte, error)
}

// Compiler defines the function type for compiling raw data into a specific type T.
type Compiler[T any] func([]byte) (T, error)

// Preloader defines the interface for identifying all available resources.
type Preloader interface {
	All(ctx context.Context) iter.Seq2[string, error]
}

// Repository manages resources of type T, providing cached access and lazy loading.
type Repository[T any] struct {
	loader   Loader
	compiler Compiler[T]
	cache    sync.Map
	sfg      singleflight.Group
}

// Registry provides an immutable snapshot of resources of type T.
type Registry[T any] struct {
	resources map[string]T
}
