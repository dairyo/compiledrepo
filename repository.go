package compiledrepo

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// call represents an in-flight request for a specific key.
type call[V any] struct {
	done chan struct{}
	val  V
	err  error
}

type panicError struct {
	value any
}

func (e *panicError) Error() string {
	return fmt.Sprintf("compilation panicked: %v", e.value)
}

// Repository manages the lifecycle of compiled resources. It provides
// efficient access to resources by combining an Opener and a Compiler,
// while implementing a caching layer and request coalescing.
type Repository[K comparable, R io.ReadCloser, V any] struct {
	opener   Opener[K, R]
	compiler Compiler[R, V]
	cache    sync.Map
	mu       sync.Mutex
	calls    map[K]*call[V]
}

// NewRepository creates a new Repository instance with the provided opener and compiler.
func NewRepository[K comparable, R io.ReadCloser, V any](opener Opener[K, R], compiler Compiler[R, V]) *Repository[K, R, V] {
	return &Repository[K, R, V]{
		opener:   opener,
		compiler: compiler,
		calls:    make(map[K]*call[V]),
	}
}

// Get retrieves a compiled resource associated with the given key.
// It first checks the cache, then uses a call-sharing mechanism to ensure
// only one compilation process occurs for the same key concurrently,
// broadcasting the result to all concurrent waiters.
func (r *Repository[K, R, V]) Get(ctx context.Context, key K) (V, error) {
	// 1. Cache Check (Fast Path)
	if val, ok := r.cache.Load(key); ok {
		return val.(V), nil
	}

	// 2. Get or Create Call Object
	r.mu.Lock()
	c, ok := r.calls[key]
	if !ok {
		c = &call[V]{
			done: make(chan struct{}),
		}
		r.calls[key] = c
	}
	r.mu.Unlock()

	// 3. Waiter Path: If another request is already in flight, wait for its result.
	if ok {
		select {
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err()
		case <-c.done:
			if e, ok := c.err.(*panicError); ok {
				panic(e.value)
			}
			return c.val, c.err
		}
	}

	// 4. Creator Path: Perform the actual work.
	defer func() {
		p := recover()
		if p != nil {
			c.err = &panicError{value: p}
		}
		close(c.done)
		r.mu.Lock()
		delete(r.calls, key)
		r.mu.Unlock()
		if p != nil {
			panic(p)
		}
	}()

	c.val, c.err = r.compileResource(ctx, key)

	return c.val, c.err
}

func (r *Repository[K, R, V]) compileResource(ctx context.Context, key K) (V, error) {
	// Double-check cache just in case
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
	val, err := r.compiler.Compile(ctx, reader)
	if err != nil {
		var zero V
		return zero, fmt.Errorf("failed to compile resource: %w", err)
	}

	// c. Cache
	r.cache.Store(key, val)

	return val, nil
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
