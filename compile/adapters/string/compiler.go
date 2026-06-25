package string

import (
	"context"
	"fmt"
	"io"

	"github.com/dairyo/compiledrepo"
)

// Compiler implements a compilation from a string representation of the input to a generic type V.
type Compiler[V any] struct{}

// NewCompiler creates a new Compiler for type V.
func NewCompiler[V any]() *Compiler[V] {
	return &Compiler[V]{}
}

// Compile reads all data from r, converts it to a string, and simulates compilation into type V.
func (c *Compiler[V]) Compile(ctx context.Context, r io.ReadCloser) (V, error) {
	var zero V

	if r == nil {
		return zero, fmt.Errorf("%w: reader is nil", compiledrepo.ErrCompile)
	}

	defer r.Close()

	// Check context before starting
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return zero, fmt.Errorf("%w: failed to read data: %w", compiledrepo.ErrCompile, err)
	}

	// Check context after potentially long read
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	content := string(data)

	// Simulate compilation from string to V.
	// In a real-world implementation, this would involve parsing the string content.
	switch any(zero).(type) {
	case string:
		return any(content).(V), nil
	case []byte:
		return any([]byte(content)).(V), nil
	default:
		return zero, fmt.Errorf("%w: unsupported target type %T for string adapter", compiledrepo.ErrCompile, zero)
	}
}
