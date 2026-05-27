package compiledrepo

import "errors"

// ErrNotFound is returned when a requested resource is not found.
var ErrNotFound = errors.New("resource not found")
