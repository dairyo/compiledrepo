package compiledrepo

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sync"
)

var (
	// ErrNotFound is returned when a resource does not exist in the filesystem.
	ErrNotFound = errors.New("resource not found")

	// ErrFiltered is returned when a resource is rejected by the PathFilter.
	ErrFiltered = errors.New("resource filtered")

	// ErrNoRuntime is returned when a resource is missing from the cache and
	// runtime compilation is disabled. This typically occurs when WithLazy()
	// is used without WithRuntime().
	ErrNoRuntime = errors.New("no runtime compile")
)

// PathNormalizer is a function type that converts an external ID into an internal file path.
type PathNormalizer func(id string) string

// PathFilter is a function type that determines whether a given path should be managed by the repository.
type PathFilter func(path string) bool

// Repository manages compiled resources of type T.
// It provides thread-safe access and supports both eager and lazy compilation strategies.
type Repository[T any] struct {
	// fsys is the source filesystem.
	fsys fs.FS
	// compiler transforms raw bytes into the compiled type T.
	compiler func([]byte) (T, error)
	// normalizer resolves IDs to clean paths.
	normalizer PathNormalizer
	// filter restricts which files are compiled into the repository.
	filter PathFilter
	// resources stores compiled instances for fast retrieval.
	resources sync.Map
	// mu protects the compilation process from race conditions during concurrent Get calls.
	mu sync.Mutex
	// runtime enables on-demand compilation during Get calls.
	runtime bool
}

// options holds the configuration for the Repository.
type options struct {
	normalizer PathNormalizer
	filter     PathFilter
	lazy       bool
	runtime    bool
}

// Option defines a functional configuration for the Repository.
type Option func(*options)

// WithNormalizer sets a custom rule for path normalization.
// By default, the ID is used directly as the path.
func WithNormalizer(n PathNormalizer) Option {
	return func(o *options) { o.normalizer = n }
}

// WithFilter sets a filter to limit which files are managed by the repository.
func WithFilter(f PathFilter) Option {
	return func(o *options) { o.filter = f }
}

// WithLazy enables lazy loading, deferring compilation until a resource is first requested via Get.
func WithLazy() Option {
	return func(o *options) { o.lazy = true }
}

// WithRuntime enables runtime compilation.
// When enabled, Get will attempt to compile and cache resources that are not yet loaded.
func WithRuntime() Option {
	return func(o *options) { o.runtime = true }
}

// New creates and initializes a new Repository for type T.
// By default, it performs eager compilation (compiling all files in the FS) unless WithLazy is provided.
func New[T any](fsys fs.FS, compiler func([]byte) (T, error), opts ...Option) (*Repository[T], error) {
	if fsys == nil || compiler == nil {
		return nil, fmt.Errorf("fsys and compiler are required")
	}

	cfg := options{
		normalizer: func(id string) string { return id },
		filter:     nil,
		lazy:       false,
		runtime:    false,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	repo := &Repository[T]{
		fsys:       fsys,
		compiler:   compiler,
		normalizer: cfg.normalizer,
		filter:     cfg.filter,
		runtime:    cfg.runtime,
	}

	// Perform eager compilation: Mutex is not required here as the instance is not
	// yet exposed to concurrent access during initialization.
	if !cfg.lazy {
		if err := repo.compileAll(); err != nil {
			return nil, fmt.Errorf("eager compilation failed: %w", err)
		}
	}
	return repo, nil
}

// compile handles the internal logic of reading, compiling, and caching a resource.
// This method is not thread-safe; callers must ensure appropriate locking.
func (r *Repository[T]) compile(key string) (T, error) {
	data, err := fs.ReadFile(r.fsys, key)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	val, err := r.compiler(data)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("compile %s: %w", key, err)
	}

	r.resources.Store(key, val)
	return val, nil
}

// compileAll scans the filesystem and compiles all files that pass the filter.
// Designed for use during initialization or controlled maintenance tasks.
func (r *Repository[T]) compileAll() error {
	return fs.WalkDir(r.fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d != nil && d.IsDir() {
			return nil
		}

		if r.filter != nil && !r.filter(p) {
			return nil
		}

		_, err = r.compile(p)
		return err
	})
}

// Get returns the compiled resource associated with the given ID.
// It is thread-safe and utilizes double-checked locking to prevent redundant
// compilation for the same resource.
func (r *Repository[T]) Get(id string) (T, error) {
	key := path.Clean(r.normalizer(id))

	if r.filter != nil && !r.filter(key) {
		var zero T
		return zero, fmt.Errorf("%w: %s", ErrFiltered, id)
	}

	// Fast Path: Return the cached resource without acquiring the lock.
	if val, ok := r.resources.Load(key); ok {
		return val.(T), nil
	}

	// Check if runtime compilation is permitted.
	if !r.runtime {
		var zero T
		return zero, fmt.Errorf("%w: %s", ErrNoRuntime, key)
	}

	// Slow Path: Protect the compilation process with a Mutex.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-checked locking: Verify if another goroutine compiled it while waiting for the lock.
	if val, ok := r.resources.Load(key); ok {
		return val.(T), nil
	}

	return r.compile(key)
}
