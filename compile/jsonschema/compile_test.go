package jsonschema

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/dairyo/compiledrepo"
)

func TestCompile_Compile(t *testing.T) {
	compiler := NewCompile[string]()

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
				tmpFile, err := os.CreateTemp("", "jsonschema-test-*.json")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())

				if _, err := tmpFile.Write([]byte(tt.content)); err != nil {
					t.Fatalf("failed to write to temp file: %v", err)
				}
				tmpFile.Seek(0, 0)

				ctx := context.Background()
				schema, err := compiler.Compile(ctx, tmpFile)

				if err != nil {
					t.Errorf("Compile() unexpected error = %v", err)
				}
				if schema == nil {
					t.Error("Compile() returned nil schema")
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
				name:    "NilFile",
				content: "",
				wantErr: compiledrepo.ErrCompile,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var tmpFile *os.File
				var err error
				if tt.name != "NilFile" {
					tmpFile, err = os.CreateTemp("", "jsonschema-test-err-*.json")
					if err != nil {
						t.Fatalf("failed to create temp file: %v", err)
					}
					defer os.Remove(tmpFile.Name())

					if _, err := tmpFile.Write([]byte(tt.content)); err != nil {
						t.Fatalf("failed to write to temp file: %v", err)
					}
					tmpFile.Seek(0, 0)
				}

				ctx := context.Background()
				_, err = compiler.Compile(ctx, tmpFile)

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
				tmpFile, err := os.CreateTemp("", "jsonschema-test-ctx-*.json")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())

				if _, err := tmpFile.Write([]byte(`{}`)); err != nil {
					t.Fatalf("failed to write to temp file: %v", err)
				}
				tmpFile.Seek(0, 0)

				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

				_, err = compiler.Compile(ctx, tmpFile)
				if !errors.Is(err, context.Canceled) {
					t.Errorf("Compile() expected context.Canceled, got %v", err)
				}
			})
		}
	})
}
