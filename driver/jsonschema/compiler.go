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
}] struct {
	configFuncs []func(*jsonschema.Compiler)
}

// NewCompiler creates a new JSON Schema compiler.
// The opts parameter accepts zero or more configuration functions that are applied
// to the underlying jsonschema.Compiler before each compilation. This allows
// users to customize settings like custom formats, content assertions, or the default draft.
func NewCompiler[R interface {
	io.Reader
	comparable
}] (opts ...func(*jsonschema.Compiler)) *Compiler[R] {
	return &Compiler[R]{
		configFuncs: opts,
	}
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
	for _, fn := range c.configFuncs {
		fn(sc)
	}
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
