package jsonschema

import (
	"encoding/json"
	"fmt"
)

// CompiledSchema is a simple representation of a compiled JSON Schema.
// In a real implementation, this would be a type from a 3rd party library.
type CompiledSchema struct {
	Raw map[string]any
}

// NewCompiler returns a Compiler that parses JSON bytes into a CompiledSchema.
func NewCompiler() func([]byte) (*CompiledSchema, error) {
	return func(data []byte) (*CompiledSchema, error) {
		var schema map[string]any
		if err := json.Unmarshal(data, &schema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal json schema: %w", err)
		}
		return &CompiledSchema{Raw: schema}, nil
	}
}
