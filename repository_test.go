package compiledrepo

import (
	"errors"
	"testing"
	"testing/fstest"
)

func TestLoad(t *testing.T) {
	const (
		testPath    = "test.txt"
		testContent = "hello world"
	)

	mockFS := fstest.MapFS{
		testPath: &fstest.MapFile{Data: []byte(testContent)},
	}

	t.Run("normal", func(t *testing.T) {
		tests := []struct {
			name     string
			compiler func([]byte) (string, error)
			want     string
		}{
			{
				name: "SuccessfulCompilation",
				compiler: func(b []byte) (string, error) {
					return string(b), nil
				},
				want: testContent,
			},
			{
				name: "TransformationInCompiler",
				compiler: func(b []byte) (string, error) {
					return "prefix_" + string(b), nil
				},
				want: "prefix_" + testContent,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     mockFS,
					compiler: tt.compiler,
				}

				val, err := repo.load(testPath)
				if err != nil {
					t.Fatalf("load() failed unexpectedly: %v", err)
				}

				// 1. Verify returned value
				if val != tt.want {
					t.Errorf("got %q, want %q", val, tt.want)
				}

				// 2. Verify side effect: check if stored in sync.Map
				cached, ok := repo.resources.Load(testPath)
				if !ok {
					t.Fatal("resource was not stored in sync.Map")
				}
				if cached.(string) != tt.want {
					t.Errorf("cached value %q, want %q", cached, tt.want)
				}
			})
		}
	})

	t.Run("error", func(t *testing.T) {
		compileErr := errors.New("compile error")

		tests := []struct {
			name     string
			path     string
			compiler func([]byte) (string, error)
			wantErr  error
		}{
			{
				name: "FileNotFound",
				path: "non-existent.txt",
				compiler: func(b []byte) (string, error) {
					return string(b), nil
				},
				wantErr: ErrNotFound,
			},
			{
				name: "CompilerReturnsError",
				path: testPath,
				compiler: func(b []byte) (string, error) {
					return "", compileErr
				},
				wantErr: compileErr,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     mockFS,
					compiler: tt.compiler,
				}

				_, err := repo.load(tt.path)
				if err == nil {
					t.Fatal("expected error, but got nil")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})
}
