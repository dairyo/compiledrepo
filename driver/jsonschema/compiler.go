package jsonschema

import (
	"context"
	"fmt"
	"io"

	"github.com/dairyo/compiledrepo"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Compiler implements the compiledrepo.Compiler interface for JSON Schemas.
type Compiler[R interface {
	io.Reader
	comparable
}] struct{}

// NewCompiler creates a new JSON Schema compiler.
func NewCompiler[R interface {
	io.Reader
	comparable
}]() *Compiler[R] {
	return &Compiler[R]{}
}

// Compile reads a JSON Schema from the provided file and "compiles" it.
// In this implementation, "compilation" is simulated by validating that the content is valid JSON.
func (c *Compiler[R]) Compile(ctx context.Context, r R) (*jsonschema.Schema, error) {

	var zeroR R
	if r == zeroR {
		return nil, fmt.Errorf("%w: reader is nil", compiledrepo.ErrCompile)
	}

	// Check for context cancellation before reading
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Compile schema
	sc := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(r)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal schema JSON: %w", compiledrepo.ErrCompile, err)
	}
	if err := sc.AddResource("schema.json", doc); err != nil {
		return nil, fmt.Errorf("%w: failed to add schema resource: %w", compiledrepo.ErrCompile, err)
	}
	sch, err := sc.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to compile schema: %w", compiledrepo.ErrCompile, err)
	}

	// Check for context cancellation after expensive operation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return sch, nil
}
