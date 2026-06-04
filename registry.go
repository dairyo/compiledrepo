package compiledrepo

import "fmt"

// Get retrieves a resource of type T associated with the given id.
//
// The returned object is considered to be part of a read-only snapshot.
// To maintain the integrity of the Registry, the internal state of the
// returned object MUST NOT be modified by the caller.
func (r *Registry[T]) Get(id string) (T, error) {
	var zero T
	res, ok := r.resources[id]
	if !ok {
		return zero, fmt.Errorf("%w: id %s", ErrNotFound, id)
	}
	return res, nil
}
