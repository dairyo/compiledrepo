package compiledrepo

import (
	"io"
	"sync"

	"golang.org/x/sync/singleflight"
)

// Repository manages the lifecycle of compiled resources. It provides
// efficient access to resources by combining an Opener and a Compiler,
// while implementing a caching layer and request coalescing.
type Repository[K comparable, R io.ReadCloser, V any] struct {
	opener   Opener[K, R]
	compiler Compiler[R, V]
	cache    sync.Map
	sfGroup  singleflight.Group
}

// NewRepository creates a new Repository instance with the provided opener and compiler.
func NewRepository[K comparable, R io.ReadCloser, V any](opener Opener[K, R], compiler Compiler[R, V]) *Repository[K, R, V] {
	return &Repository[K, R, V]{
		opener:   opener,
		compiler: compiler,
		// cache and sfGroup are zero-value initialized, which is correct for sync.Map and singleflight.Group.
	}
}
