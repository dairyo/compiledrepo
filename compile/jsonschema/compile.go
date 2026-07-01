package jsonschema

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dairyo/compiledrepo"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validate represents a compiled JSON resource with its metadata and schema.
type Validate[T any] struct {
	Metadata T
	Schema   *jsonschema.Schema
}

// Compile implements the compiledrepo.Compiler interface for JSON Schemas.
type Compile[T any] struct{}

// NewCompile creates a new JSON Schema compiler.
func NewCompile[T any]() *Compile[T] {
	return &Compile[T]{}
}

// Compile reads a JSON Schema from the provided file and "compiles" it.
// In this implementation, "compilation" is simulated by validating that the content is valid JSON.
func (c *Compile[T]) Compile(ctx context.Context, f *os.File) (*Schema, error) {
	if f == nil {
		return nil, fmt.Errorf("%w: file is nil", compiledrepo.ErrCompile)
	}

	// Check for context cancellation before reading
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Read the entire content of the file
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read file: %w", compiledrepo.ErrCompile, err)
	}

	// Simulate compilation by validating JSON
	if !json.Valid(data) {
		return nil, fmt.Errorf("%w: invalid JSON format", compiledrepo.ErrCompile)
	}

	// Check for context cancellation after expensive operation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &Schema{Raw: data}, nil
}
