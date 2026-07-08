package jsonschema

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/dairyo/compiledrepo"
)

func TestCompiler_Compile(t *testing.T) {
	compiler := NewCompiler[io.Reader]()

	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name    string
			content string
		}{
			{
				name:    "ValidJSON",
				content: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			},
			{
				name:    "EmptyJSON",
				content: `{}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := context.Background()
				r := io.NopCloser(bytes.NewReader([]byte(tt.content)))
				val, err := compiler.Compile(ctx, r)

				if err != nil {
					t.Fatalf("Compile() unexpected error = %v", err)
				}
				if val == nil {
					t.Fatal("Compile() returned nil result")
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name    string
			content string
			wantErr error
		}{
			{
				name:    "InvalidJSON",
				content: `{"type": "object", "properties": { "name": "string"`, // Missing closing brace
				wantErr: compiledrepo.ErrCompile,
			},
			{
				name:    "InvalidSchema",
				content: `{"type": "not-a-type"}`,
				wantErr: compiledrepo.ErrCompile,
			},
			{
				name:    "NilReader",
				content: "",
				wantErr: compiledrepo.ErrCompile,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var r io.Reader
				if tt.name != "NilReader" {
					r = io.NopCloser(bytes.NewReader([]byte(tt.content)))
				}

				ctx := context.Background()
				_, err := compiler.Compile(ctx, r)

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Compile() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		tests := []struct {
			name string
		}{
			{
				name: "CancelledContext",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := io.NopCloser(bytes.NewReader([]byte(`{`)))

				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

				_, err := compiler.Compile(ctx, r)
				if !errors.Is(err, context.Canceled) {
					t.Errorf("Compile() expected %v, got %v", context.Canceled, err)
				}
			})
		}
	})
}
