package compiledrepo

import "errors"

// ErrInvalidArg is returned when an invalid argument is provided,
// such as a nil dependency during initialization.
var ErrInvalidArg = errors.New("invalid argument")

// ErrNotFound is returned when a resource cannot be found with the specified ID.
var ErrNotFound = errors.New("resource not found")
