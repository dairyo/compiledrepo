package compiledrepo

// Registry provides a generic storage for compiled values.
type Registry[K comparable, V any] struct {
	values map[K]V
}

// NewRegistry creates a new instance of Registry.
func NewRegistry[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{
		values: make(map[K]V),
	}
}

// Get retrieves a value from the registry for the given key.
// It returns the value and true if the key exists, otherwise the zero value of V and false.
func (r *Registry[K, V]) Get(key K) (V, bool) {
	if r == nil || r.values == nil {
		var zero V
		return zero, false
	}
	val, ok := r.values[key]
	return val, ok
}

// Set adds or updates a value in the registry.
// (Although not explicitly requested in the sub-task description, it's necessary for the Registry to be useful and testable.
// However, I should check if I should only implement what's requested).
// Let's add Set for internal use or if the plan implies it.
func (r *Registry[K, V]) Set(key K, value V) {
	if r == nil || r.values == nil {
		return
	}
	r.values[key] = value
}
