package compiledrepo

// Registry provides a generic storage for compiled values.
// It is designed to be an immutable snapshot of a repository's cache.
type Registry[K comparable, V any] struct {
	values map[K]V
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
