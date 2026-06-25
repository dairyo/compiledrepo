package jsonschema

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dairyo/compiledrepo"
)

// Schema represents a compiled JSON Schema.
type Schema struct {
	Raw []byte
}

// Compiler implements the compiledrepo.Compiler interface for JSON Schemas.
type Compiler struct{}

// NewCompiler creates a new JSON Schema compiler.
func NewCompiler() *Compiler {
	return &Compiler{}
}

// Compile reads a JSON Schema from the provided file and "compiles" it.
// In this implementation, "compilation" is simulated by validating that the content is valid JSON.
func (c *Compiler) Compile(ctx context.Context, f *os.File) (*Schema, error) {
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
