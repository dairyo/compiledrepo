package compiledrepo

// Get retrieves a resource from the registry.
//
// Note: Since the Registry may contain reference types (pointers, maps, slices),
// callers MUST NOT mutate the returned object to ensure the integrity of the
// snapshot. Mutating the returned object may lead to unexpected behavior in
// other parts of the application sharing the same Registry.
func (r *Registry[T]) Get(id string) (T, error) {
	val, ok := r.resources[id]
	if !ok {
		var zero T
		return zero, ErrNotFound
	}
	return val, nil
}
