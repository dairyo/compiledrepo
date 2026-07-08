package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/dairyo/compiledrepo"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validate represents a compiled JSON resource with its schema.
type Validate[T any] struct {
	Schema   *jsonschema.Schema
}

type envelope struct {
	Schema   json.RawMessage `json:"schema"`
}

// Compiler implements the compiledrepo.Compiler interface for JSON Schemas.
type Compiler[T any, R interface {
	io.Reader
	comparable
}] struct{}

// NewCompiler creates a new JSON Schema compiler.
func NewCompiler[T any, R interface {
	io.Reader
	comparable
}]() *Compiler[T, R] {
	return &Compiler[T, R]{}
}

// Compile reads a JSON Schema from the provided file and "compiles" it.
// In this implementation, "compilation" is simulated by validating that the content is valid JSON.
func (c *Compiler[T, R]) Compile(ctx context.Context, r R) (*Validate[T], error) {

	var zeroR R
	if r == zeroR {
		return nil, fmt.Errorf("%w: reader is nil", compiledrepo.ErrCompile)
	}

	// Check for context cancellation before reading
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Read the entire content of the file
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read file: %w", compiledrepo.ErrCompile, err)
	}

	// Parse into envelope
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON format: %w", compiledrepo.ErrCompile, err)
	}

	if len(env.Schema) == 0 {
		return nil, fmt.Errorf("%w: missing schema field", compiledrepo.ErrCompile)
	}

	// Compile schema
	sc := jsonschema.NewCompiler()
	if err := sc.AddResource("schema.json", bytes.NewReader(env.Schema)); err != nil {
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

	return &Validate[T]{
		Schema:   sch,
	}, nil
}
