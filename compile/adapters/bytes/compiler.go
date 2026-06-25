package bytes

import (
	"context"
	"fmt"
	"io"

	"github.com/dairyo/compiledrepo"
)

// Compiler implements a simple compilation from bytes to a generic type V.
type Compiler[V any] struct{}

// NewCompiler creates a new Compiler for type V.
func NewCompiler[V any]() *Compiler[V] {
	return &Compiler[V]{}
}

// Compile reads all data from r and simulates compilation into type V.
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

	// Simulate compilation based on target type V
	switch any(zero).(type) {
	case []byte:
		return any(data).(V), nil
	case string:
		return any(string(data)).(V), nil
	default:
		return zero, fmt.Errorf("%w: unsupported target type %T", compiledrepo.ErrCompile, zero)
	}
}
