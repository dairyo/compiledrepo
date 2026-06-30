package compiledrepo

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// Repository manages the lifecycle of compiled resources. It provides
// efficient access to resources by combining an Opener and a Compiler,
// while implementing a caching layer.
type Repository[K comparable, R io.ReadCloser, V any] struct {
	opener   Opener[K, R]
	compiler Compiler[R, V]
	cache    sync.Map
	mu       sync.Mutex
	muxMap   map[K]*sync.Mutex
}

// NewRepository creates a new Repository instance with the provided opener and compiler.
func NewRepository[K comparable, R io.ReadCloser, V any](opener Opener[K, R], compiler Compiler[R, V]) *Repository[K, R, V] {
	return &Repository[K, R, V]{
		opener:   opener,
		compiler: compiler,
		muxMap:   make(map[K]*sync.Mutex),
	}
}

// Get retrieves a compiled resource associated with the given key.
// It first checks the cache, then uses a mutex to ensure only one
// compilation process occurs at a time, with a double-check of the cache.
func (r *Repository[K, R, V]) Get(ctx context.Context, key K) (V, error) {
	// 1. Cache Check (Fast Path)
	if val, ok := r.cache.Load(key); ok {
		return val.(V), nil
	}

	// 2. Lock for Compilation
	val, ok, mux := func() (V, bool, *sync.Mutex) {
		r.mu.Lock()
		defer r.mu.Unlock()
		if val, ok := r.cache.Load(key); ok {
			return val.(V), true, nil
		}
		if mux, ok := r.muxMap[key]; ok {
			var zero V
			return zero, false, mux
		}
		mux := &sync.Mutex{}
		r.muxMap[key] = mux
		var zero V
		return zero, false, mux
	}()
	if ok {
		return val, nil
	}
	mux.Lock()
	defer mux.Unlock()

	// Double-check cache after acquiring lock to avoid redundant compilation
	if val, ok := r.cache.Load(key); ok {
		return val.(V), nil
	}

	// a. Open
	reader, err := r.opener.Open(ctx, key)
	if err != nil {
		var zero V
		return zero, fmt.Errorf("failed to open resource: %w", err)
	}
	defer reader.Close()

	// b. Compile
	compiled, err := r.compiler.Compile(ctx, reader)
	if err != nil {
		var zero V
		return zero, fmt.Errorf("failed to compile resource: %w", err)
	}

	// c. Cache
	r.cache.Store(key, compiled)

	return compiled, nil
}

// Preload populates the cache with resources whose keys are provided by the KeyIterator.
// It iterates through all keys and calls Get for each. If any error occurs during
// iteration or retrieval, Preload stops and returns the error.
func (r *Repository[K, R, V]) Preload(ctx context.Context, it KeyIterator[K]) error {
	if it == nil {
		return fmt.Errorf("%w: iterator is nil", ErrIterator)
	}
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

// Snapshot creates an immutable snapshot of the current repository cache.
// The returned Registry represents the state of the cache at the time of the call.
func (r *Repository[K, R, V]) Snapshot() Registry[K, V] {
	snapshotMap := make(map[K]V)
	r.cache.Range(func(key, value any) bool {
		snapshotMap[key.(K)] = value.(V)
		return true
	})
	return Registry[K, V]{
		values: snapshotMap,
	}
}
