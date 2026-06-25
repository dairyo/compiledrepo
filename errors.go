package compiledrepo

import "errors"

var (
	// ErrOpen is returned when the file or resource cannot be opened.
	ErrOpen = errors.New("failed to open resource")
	// ErrCompile is returned when the compilation process fails.
	ErrCompile = errors.New("failed to compile")
	// ErrIterator is returned when an error occurs during iteration.
	ErrIterator = errors.New("iterator error")
)
