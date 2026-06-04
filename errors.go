package compiledrepo

import "errors"

var (
	// ErrInvalidArg is returned when an invalid argument is provided to a function.
	ErrInvalidArg = errors.New("invalid argument")
	// ErrNotFound is returned when a requested resource cannot be found.
	ErrNotFound = errors.New("resource not found")
)
