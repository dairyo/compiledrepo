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
)

// PathNormalizer is a function type that converts an external ID into an internal file path.
type PathNormalizer func(id string) string

// PathFilter is a function type that determines whether a given path should be managed by the repository.
type PathFilter func(path string) bool

// Repository manages compiled resources of type T.
// It provides thread-safe access and optional eager loading.
type Repository[T any] struct {
	// fsys is the source filesystem.
	fsys fs.FS
	// compiler is a function that transforms raw bytes into type T.
	compiler func([]byte) (T, error)
	// normalizer resolves IDs to clean paths.
	normalizer PathNormalizer
	// filter restricts which files are loaded.
	filter PathFilter
	// resources stores compiled instances.
	resources sync.Map
	// mu protects the loading process from race conditions during concurrent Get calls.
	mu sync.Mutex
}

// options holds configuration for the Repository.
type options struct {
	normalizer PathNormalizer
	filter     PathFilter
	lazy       bool
}

// Option defines a functional configuration for the Repository.
type Option func(*options)

// WithNormalizer sets a custom rule for path normalization.
func WithNormalizer(n PathNormalizer) Option {
	return func(o *options) { o.normalizer = n }
}

// WithFilter sets a filter to limit which files are loaded into the repository.
func WithFilter(f PathFilter) Option {
	return func(o *options) { o.filter = f }
}

// WithLazy enables lazy loading, deferring compilation until the resource is first requested.
func WithLazy() Option {
	return func(o *options) { o.lazy = true }
}

// New creates and initializes a new Repository for type T.
// By default, it performs eager loading unless WithLazy is provided.
func New[T any](fsys fs.FS, compiler func([]byte) (T, error), opts ...Option) (*Repository[T], error) {
	if fsys == nil || compiler == nil {
		return nil, fmt.Errorf("fsys and compiler are required")
	}

	cfg := options{
		normalizer: func(id string) string { return id },
		filter:     nil,
		lazy:       false,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	repo := &Repository[T]{
		fsys:       fsys,
		compiler:   compiler,
		normalizer: cfg.normalizer,
		filter:     cfg.filter,
	}

	// Perform eager loading: Lock is omitted here as the instance is not yet exposed
	// to external concurrent access during initialization.
	if !cfg.lazy {
		if err := repo.compileAll(); err != nil {
			return nil, fmt.Errorf("eager load failed: %w", err)
		}
	}
	return repo, nil
}

// load handles the core logic of reading, compiling, and storing a resource.
// It does not handle synchronization; callers must manage locks if necessary.
func (r *Repository[T]) load(key string) (T, error) {
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

// compileAll scans the filesystem and compiles all valid resources.
// It is intended for use during initialization where thread safety is guaranteed by the caller.
func (r *Repository[T]) compileAll() error {
	return fs.WalkDir(r.fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if r.filter != nil && !r.filter(p) {
			return nil
		}

		_, err = r.load(p)
		return err
	})
}

// Get returns the compiled resource associated with the given ID.
// It is thread-safe and implements double-checked locking to ensure only one compilation occurs per ID.
func (r *Repository[T]) Get(id string) (T, error) {
	key := path.Clean(r.normalizer(id))

	if r.filter != nil && !r.filter(key) {
		var zero T
		return zero, fmt.Errorf("%w: %s", ErrFiltered, id)
	}

	// Fast Path: Return already loaded resource without locking.
	if val, ok := r.resources.Load(key); ok {
		return val.(T), nil
	}

	// Slow Path: Protect the compilation process with a Mutex.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-checked locking to prevent redundant loads.
	if val, ok := r.resources.Load(key); ok {
		return val.(T), nil
	}

	return r.load(key)
}
