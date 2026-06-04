package compiledrepo

import "errors"

// ErrInvalidArg is returned when a provided argument is invalid (e.g., nil or empty).
var ErrInvalidArg = errors.New("invalid argument")

// ErrNotFound is returned when a requested resource cannot be found.
var ErrNotFound = errors.New("resource not found")
