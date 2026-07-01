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

// Validate represents a compiled JSON resource with its metadata and schema.
type Validate[T any] struct {
	Metadata T
	Schema   *jsonschema.Schema
}

type envelope struct {
	Metadata json.RawMessage `json:"metadata"`
	Schema   json.RawMessage `json:"schema"`
}

// Compile implements the compiledrepo.Compiler interface for JSON Schemas.
type Compile[T any] struct{}

// NewCompile creates a new JSON Schema compiler.
func NewCompile[T any]() *Compile[T] {
	return &Compile[T]{}
}

// Compile reads a JSON Schema from the provided file and "compiles" it.
// In this implementation, "compilation" is simulated by validating that the content is valid JSON.
func (c *Compile[T]) Compile(ctx context.Context, r io.ReadCloser) (*Validate[T], error) {
	if r == nil {
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

	// Extract metadata
	var metadata T
	if err := json.Unmarshal(env.Metadata, &metadata); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal metadata: %w", compiledrepo.ErrCompile, err)
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
		Metadata: metadata,
		Schema:   sch,
	}, nil
}
