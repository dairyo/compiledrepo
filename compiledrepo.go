package compiledrepo

import (
	"context"
	"iter"
)

// Loader is responsible for loading raw bytes of a resource by its identifier.
type Loader interface {
	Load(ctx context.Context, id string) ([]byte, error)
}

// Compiler is a function that transforms raw bytes into a typed resource.
type Compiler[T any] func([]byte) (T, error)

// Preloader provides a sequence of resource identifiers available for preloading.
type Preloader interface {
	All(ctx context.Context) iter.Seq2[string, error]
}

// Repository manages the loading, compiling, and caching of resources.
type Repository[T any] struct {
	loader   Loader
	compiler Compiler[T]
	cache    syncMap // Internal cache for compiled resources.
	sfg      singleflightGroup
}

// Registry provides a read-only snapshot of the compiled resources.
type Registry[T any] struct {
	resources map[string]T
}

// Internal types for encapsulation and dependency management.
type syncMap interface {
	Load(key any) (value any, loaded bool)
	Store(key, value any)
	Range(f func(key, value any) bool)
}

type singleflightGroup interface {
	Do(key string, fn func() (interface{}, error)) (interface{}, error, bool)
}
