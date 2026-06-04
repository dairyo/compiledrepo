package compiledrepo

import (
	"errors"
)

// ErrInvalidArg is returned when an input argument is empty or invalid.
var ErrInvalidArg = errors.New("invalid argument")

// ErrNotFound is returned when a requested resource cannot be found.
var ErrNotFound = errors.New("resource not found")
