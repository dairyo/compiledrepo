package compiledrepo

import (
	"context"
	"fmt"
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
	}
}

// Get retrieves a compiled resource associated with the given key.
// It first checks the cache, then uses request coalescing to ensure only one
// compilation process occurs for the same key concurrently.
func (r *Repository[K, R, V]) Get(ctx context.Context, key K) (v V, err error) {
	// Panic recovery to ensure the repository doesn't crash the application
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered during Get: %v", p)
		}
	}()

	// 1. Cache Check
	if val, ok := r.cache.Load(key); ok {
		return val.(V), nil
	}

	// 2. Request Coalescing
	// singleflight.Group.Do requires a string key.
	keyStr := fmt.Sprintf("%v", key)
	res, sfErr, _ := r.sfGroup.Do(keyStr, func() (any, error) {
		// Double check cache inside the coalescing function to avoid redundant work
		if val, ok := r.cache.Load(key); ok {
			return val, nil
		}

		// a. Open
		reader, err := r.opener.Open(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to open resource: %w", err)
		}
		defer reader.Close()

		// b. Compile
		compiled, err := r.compiler.Compile(ctx, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to compile resource: %w", err)
		}

		// c. Cache
		r.cache.Store(key, compiled)
		return compiled, nil
	})

	if sfErr != nil {
		return v, sfErr
	}

	return res.(V), nil
}

// Preload populates the cache with resources whose keys are provided by the KeyIterator.
// It iterates through all keys and calls Get for each. If any error occurs during
// iteration or retrieval, Preload stops and returns the error.
func (r *Repository[K, R, V]) Preload(ctx context.Context, it KeyIterator[K]) error {
	for key, err := range it.All(ctx) {
		if err != nil {
			return fmt.Errorf("preload iteration failed: %w", err)
		}

		if _, err := r.Get(ctx, key); err != nil {
			return fmt.Errorf("preload failed for key %v: %w", key, err)
		}

		// Check for context cancellation to ensure responsiveness
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}
