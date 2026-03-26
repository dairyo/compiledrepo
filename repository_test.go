package compiledrepo

import (
	"errors"
	"io/fs"
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

func TestCompileAll(t *testing.T) {
	// Standard compiler for normal cases
	mockCompiler := func(b []byte) (string, error) {
		return string(b), nil
	}

	// --- Normal Cases ---
	t.Run("normal", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"root.txt":    &fstest.MapFile{Data: []byte("root_data")},
			"sub":         &fstest.MapFile{Mode: fs.ModeDir},
			"sub/dir.txt": &fstest.MapFile{Data: []byte("sub_data")},
			"skip.tmp":    &fstest.MapFile{Data: []byte("ignore")},
		}

		tests := []struct {
			name     string
			filter   PathFilter
			expected map[string]string // path -> expected compiled value
		}{
			{
				name:   "LoadFilesRecursively",
				filter: nil,
				expected: map[string]string{
					"root.txt":    "root_data",
					"sub/dir.txt": "sub_data",
					"skip.tmp":    "ignore",
				},
			},
			{
				name: "ApplyFilter",
				filter: func(p string) bool {
					return p != "skip.tmp"
				},
				expected: map[string]string{
					"root.txt":    "root_data",
					"sub/dir.txt": "sub_data",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     mockFS,
					compiler: mockCompiler,
					filter:   tt.filter,
				}

				if err := repo.compileAll(); err != nil {
					t.Fatalf("compileAll() failed: %v", err)
				}

				// Verify each expected file content
				for path, want := range tt.expected {
					got, ok := repo.resources.Load(path)
					if !ok {
						t.Errorf("path %q was not found in resources", path)
						continue
					}
					if got.(string) != want {
						t.Errorf("path %q: got %q, want %q", path, got, want)
					}
				}

				// Verify total count (ensures no directories or filtered files are stored)
				count := 0
				repo.resources.Range(func(_, _ any) bool {
					count++
					return true
				})
				if count != len(tt.expected) {
					t.Errorf("resource count: got %d, want %d", count, len(tt.expected))
				}
			})
		}
	})

	// --- Error Cases ---
	t.Run("error", func(t *testing.T) {
		compErr := errors.New("compile error")

		tests := []struct {
			name     string
			fsys     fs.FS
			compiler func([]byte) (string, error)
			wantErr  error
		}{
			{
				name: "StopOnCompilationError",
				fsys: fstest.MapFS{
					"root.txt": &fstest.MapFile{Data: []byte("fail")},
				},
				compiler: func(b []byte) (string, error) {
					return "", compErr
				},
				wantErr: compErr,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     tt.fsys,
					compiler: tt.compiler,
				}

				err := repo.compileAll()
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got error %v, want %v", err, tt.wantErr)
				}
			})
		}
	})
}
