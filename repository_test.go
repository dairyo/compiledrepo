package compiledrepo

import (
	"errors"
	"io/fs"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

func TestWithNormalizer(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		fn       PathNormalizer
		expected string
	}{
		{
			name:     "identity mapping",
			id:       "file.txt",
			fn:       func(id string) string { return id },
			expected: "file.txt",
		},
		{
			name:     "add directory",
			id:       "file.txt",
			fn:       func(id string) string { return "templates/" + id },
			expected: "templates/file.txt",
		},
		{
			name:     "add extension",
			id:       "file",
			fn:       func(id string) string { return id + ".json" },
			expected: "file.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options{}
			WithNormalizer(tt.fn)(opts)
			if opts.normalizer == nil {
				t.Fatal("normalizer should not be nil")
			}
			if got := opts.normalizer(tt.id); got != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestWithFilter(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		fn       PathFilter
		expected bool
	}{
		{
			name:     "allow only .txt files",
			path:     "document.txt",
			fn:       func(path string) bool { return strings.HasSuffix(path, ".txt") },
			expected: true,
		},
		{
			name:     "reject .json files",
			path:     "config.json",
			fn:       func(path string) bool { return !strings.HasSuffix(path, ".json") },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options{}
			WithFilter(tt.fn)(opts)
			if opts.filter == nil {
				t.Fatal("filter should not be nil")
			}
			if got := opts.filter(tt.path); got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWithMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     Mode
		expected Mode
	}{
		{"Eager", Eager, Eager},
		{"Lazy", Lazy, Lazy},
		{"EagerWithRuntime", EagerWithRuntime, EagerWithRuntime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options{}
			WithMode(tt.mode)(opts)
			if opts.mode != tt.expected {
				t.Errorf("got %v, want %v", opts.mode, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	successCompiler := func(b []byte) (string, error) { return string(b), nil }
	failCompiler := func(b []byte) (string, error) { return "", errors.New("compile fail") }
	fsys := fstest.MapFS{"test.txt": {Data: []byte("hello")}}

	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			name string
			mode Mode
		}{
			{"Eager", Eager},
			{"Lazy", Lazy},
			{"EagerWithRuntime", EagerWithRuntime},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo, err := New[string](fsys, successCompiler, WithMode(tt.mode))
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if repo == nil {
					t.Fatal("repository should not be nil")
				}
			})
		}
	})

	t.Run("Failure", func(t *testing.T) {
		tests := []struct {
			name     string
			fsys     fs.FS
			compiler func([]byte) (string, error)
			opts     []Option
		}{
			{
				name:     "nil fsys",
				fsys:     nil,
				compiler: successCompiler,
				opts:     nil,
			},
			{
				name:     "nil compiler",
				fsys:     fsys,
				compiler: nil,
				opts:     nil,
			},
			{
				name:     "invalid mode",
				fsys:     fsys,
				compiler: successCompiler,
				opts:     []Option{WithMode(Mode(999))},
			},
			{
				name:     "eager compilation failure",
				fsys:     fsys,
				compiler: failCompiler,
				opts:     []Option{WithMode(Eager)},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo, err := New[string](tt.fsys, tt.compiler, tt.opts...)
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if repo != nil {
					t.Errorf("expected nil repository on error, got %v", repo)
				}
			})
		}
	})
}

func TestRepository_Get(t *testing.T) {
	compiler := func(b []byte) (string, error) { return string(b), nil }
	fsys := fstest.MapFS{
		"a.txt": {Data: []byte("content-a")},
		"b.txt": {Data: []byte("content-b")},
	}

	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			name       string
			mode       Mode
			id         string
			normalizer PathNormalizer
			expected   string
		}{
			{
				name:     "Fast Path (Cache Hit)",
				mode:     Eager,
				id:       "a.txt",
				expected: "content-a",
			},
			{
				name:     "Slow Path (Lazy Load)",
				mode:     Lazy,
				id:       "a.txt",
				expected: "content-a",
			},
			{
				name: "Normalizer and Clean",
				mode: Lazy,
				id:   "a.txt",
				normalizer: func(id string) string {
					return id + "/../a.txt"
				},
				expected: "content-a",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var opts []Option
				opts = append(opts, WithMode(tt.mode))
				if tt.normalizer != nil {
					opts = append(opts, WithNormalizer(tt.normalizer))
				}

				repo, _ := New[string](fsys, compiler, opts...)
				val, err := repo.Get(tt.id)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if val != tt.expected {
					t.Errorf("got %s, want %s", val, tt.expected)
				}
			})
		}
	})

	t.Run("Failure", func(t *testing.T) {
		tests := []struct {
			name     string
			mode     Mode
			id       string
			filter   PathFilter
			expected error
		}{
			{
				name: "Filtered",
				mode: Eager,
				id:   "a.txt",
				filter: func(p string) bool {
					return p != "a.txt"
				},
				expected: ErrFiltered,
			},
			{
				name:     "No Runtime",
				mode:     Eager,
				id:       "non-existent.txt",
				expected: ErrNoRuntime,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var opts []Option
				opts = append(opts, WithMode(tt.mode))
				if tt.filter != nil {
					opts = append(opts, WithFilter(tt.filter))
				}

				repo, _ := New[string](fsys, compiler, opts...)
				_, err := repo.Get(tt.id)
				if !errors.Is(err, tt.expected) {
					t.Errorf("expected error %v, got %v", tt.expected, err)
				}
			})
		}
	})
}

func TestRepository_compile(t *testing.T) {
	successCompiler := func(b []byte) (string, error) { return string(b), nil }
	errInternal := errors.New("internal error")
	failCompiler := func(b []byte) (string, error) { return "", errInternal }

	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			name        string
			fs          fstest.MapFS
			key         string
			expectedVal string
		}{
			{
				name: "standard file",
				fs: fstest.MapFS{
					"hello.txt": {Data: []byte("hello world")},
				},
				key:         "hello.txt",
				expectedVal: "hello world",
			},
			{
				name: "empty file",
				fs: fstest.MapFS{
					"empty.txt": {Data: []byte("")},
				},
				key:         "empty.txt",
				expectedVal: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     tt.fs,
					compiler: successCompiler,
				}

				val, err := repo.compile(tt.key)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if val != tt.expectedVal {
					t.Errorf("got %s, want %s", val, tt.expectedVal)
				}
				if cached, ok := repo.resources.Load(tt.key); !ok || cached.(string) != tt.expectedVal {
					t.Error("value was not stored in resources map")
				}
			})
		}
	})

	t.Run("Failure", func(t *testing.T) {
		tests := []struct {
			name                string
			fs                  fstest.MapFS
			key                 string
			compiler            func([]byte) (string, error)
			expectedError       error
			expectedInternalErr error
		}{
			{
				name:          "filesystem error",
				fs:            fstest.MapFS{},
				key:           "missing.txt",
				compiler:      successCompiler,
				expectedError: ErrNotFound,
			},
			{
				name: "compiler error",
				fs: fstest.MapFS{
					"error.txt": {Data: []byte("some data")},
				},
				key:                 "error.txt",
				compiler:            failCompiler,
				expectedError:       ErrCompile,
				expectedInternalErr: errInternal,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     tt.fs,
					compiler: tt.compiler,
				}

				_, err := repo.compile(tt.key)
				if err == nil {
					t.Fatal("expected error but got nil")
				}

				if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error to wrap %v, got %v", tt.expectedError, err)
				}

				if tt.expectedInternalErr != nil && !errors.Is(err, tt.expectedInternalErr) {
					t.Errorf("expected error to wrap internal error %v, got %v", tt.expectedInternalErr, err)
				}
			})
		}
	})
}

type mockErrorFS struct {
	fs.FS
}

func (m *mockErrorFS) Open(name string) (fs.File, error) {
	return nil, errors.New("filesystem read error")
}

func TestRepository_compileAll(t *testing.T) {
	successCompiler := func(b []byte) (string, error) { return string(b), nil }
	failCompiler := func(b []byte) (string, error) { return "", errors.New("compile fail") }

	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			name             string
			fsys             fstest.MapFS
			filter           PathFilter
			expectedCompiled map[string]string
			expectedIgnored  []string
		}{
			{
				name: "nested directories and filtering",
				fsys: fstest.MapFS{
					"root.txt":            {Data: []byte("root-content")},
					"dir1/file1.txt":      {Data: []byte("f1-content")},
					"dir1/dir2/file2.txt": {Data: []byte("f2-content")},
					"dir1/ignore.json":    {Data: []byte("ignore-content")},
					"dir2/ignore.log":     {Data: []byte("ignore-content")},
				},
				filter: func(path string) bool {
					return strings.HasSuffix(path, ".txt")
				},
				expectedCompiled: map[string]string{
					"root.txt":            "root-content",
					"dir1/file1.txt":      "f1-content",
					"dir1/dir2/file2.txt": "f2-content",
				},
				expectedIgnored: []string{"dir1/ignore.json", "dir2/ignore.log"},
			},
			{
				name: "no files in fs",
				fsys: fstest.MapFS{},
				filter: func(path string) bool {
					return true
				},
				expectedCompiled: map[string]string{},
				expectedIgnored:  []string{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     tt.fsys,
					compiler: successCompiler,
					filter:   tt.filter,
				}

				if err := repo.compileAll(); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				for path, expectedVal := range tt.expectedCompiled {
					val, ok := repo.resources.Load(path)
					if !ok {
						t.Errorf("expected file %s to be compiled, but it was not", path)
						continue
					}
					if gotVal := val.(string); gotVal != expectedVal {
						t.Errorf("file %s: got %s, want %s", path, gotVal, expectedVal)
					}
				}

				for _, path := range tt.expectedIgnored {
					if _, ok := repo.resources.Load(path); ok {
						t.Errorf("expected file %s to be ignored, but it was compiled", path)
					}
				}
			})
		}
	})

	t.Run("Failure", func(t *testing.T) {
		tests := []struct {
			name     string
			fsys     fs.FS
			compiler func([]byte) (string, error)
			expected error
		}{
			{
				name:     "walk error",
				fsys:     &mockErrorFS{},
				compiler: successCompiler,
				expected: ErrWalk,
			},
			{
				name: "compile error during all",
				fsys: fstest.MapFS{
					"bad.txt": {Data: []byte("data")},
				},
				compiler: failCompiler,
				expected: ErrCompile,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     tt.fsys,
					compiler: tt.compiler,
					filter:   func(p string) bool { return true },
				}

				err := repo.compileAll()
				if !errors.Is(err, tt.expected) {
					t.Errorf("expected error %v, got %v", tt.expected, err)
				}
			})
		}
	})
}

func TestRepository_Concurrency(t *testing.T) {
	var compileCount int32
	compiler := func(b []byte) (string, error) {
		atomic.AddInt32(&compileCount, 1)
		return string(b), nil
	}
	fsys := fstest.MapFS{"shared.txt": {Data: []byte("shared")}}
	repo, _ := New[string](fsys, compiler, WithMode(Lazy))

	var wg sync.WaitGroup
	numGoroutines := 100
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = repo.Get("shared.txt")
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&compileCount) != 1 {
		t.Errorf("compiler should be called exactly once, got %d", compileCount)
	}
}
