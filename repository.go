package compiledrepo

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
)

// NewRepository creates a new Repository with the given loader and compiler.
func NewRepository[T any](loader Loader, compiler Compiler[T]) *Repository[T] {
	return &Repository[T]{
		loader:   loader,
		compiler: compiler,
		cache:    &sync.Map{},
		sfg:      &singleflight.Group{},
	}
}

// Get retrieves a resource by its ID. It uses a cache and ensures that
// only one load/compile operation happens concurrently for the same ID.
func (r *Repository[T]) Get(ctx context.Context, id string) (T, error) {
	var zero T

	// 1. Check cache
	if val, ok := r.cache.Load(id); ok {
		return val.(T), nil
	}

	// 2. Deduplicate concurrent loads
	val, err, _ := r.sfg.Do(id, func() (interface{}, error) {
		// Check cache again inside singleflight to avoid redundant loads
		if val, ok := r.cache.Load(id); ok {
			return val, nil
		}

		// Load
		data, err := r.loader.Load(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("loader failed for %s: %w", id, err)
		}

		// Compile
		compiled, err := r.compiler(data)
		if err != nil {
			return nil, fmt.Errorf("compiler failed for %s: %w", id, err)
		}

		// Cache
		r.cache.Store(id, compiled)
		return compiled, nil
	})

	if err != nil {
		return zero, err
	}

	return val.(T), nil
}

// Preload explicitly fills the cache by iterating over the provided Preloader.
func (r *Repository[T]) Preload(ctx context.Context, p Preloader) error {
	for id, err := range p.All(ctx) {
		// Explicitly check for context cancellation to stop preloading immediately
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}
		if _, err := r.Get(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// Snapshot creates a Registry which is a static copy of the current cache.
func (r *Repository[T]) Snapshot() *Registry[T] {
	res := make(map[string]T)
	r.cache.Range(func(key, value any) bool {
		res[key.(string)] = value.(T)
		return true
	})
	return &Registry[T]{resources: res}
}
