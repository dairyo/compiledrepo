package compiledrepo

import (
	"errors"
	"io/fs"
	"strings"
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
		{
			name:     "add directory and extension",
			id:       "file",
			fn:       func(id string) string { return "template/" + id + ".json" },
			expected: "template/file.json",
		},
		{
			name:     "ChangeExtension",
			id:       "data.json",
			fn:       func(id string) string { return id[:len(id)-4] + "yaml" },
			expected: "data.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options{}
			opt := WithNormalizer(tt.fn)
			opt(opts)

			if opts.normalizer == nil {
				t.Fatal("normalizer should not be nil")
			}

			got := opts.normalizer(tt.id)
			if got != tt.expected {
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
			name:     "always allow",
			path:     "file.txt",
			fn:       func(path string) bool { return true },
			expected: true,
		},
		{
			name:     "always reject",
			path:     "file.txt",
			fn:       func(path string) bool { return false },
			expected: false,
		},
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
		{
			name:     "allow only files in assets directory",
			path:     "assets/image.png",
			fn:       func(path string) bool { return strings.HasPrefix(path, "assets/") },
			expected: true,
		},
		{
			name:     "reject files outside assets directory",
			path:     "src/main.go",
			fn:       func(path string) bool { return strings.HasPrefix(path, "assets/") },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options{}
			opt := WithFilter(tt.fn)
			opt(opts)

			if opts.filter == nil {
				t.Fatal("filter should not be nil")
			}

			got := opts.filter(tt.path)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWithMode(t *testing.T) {
	tests := []struct {
		name            string
		mode            Mode
		expectedLazy    bool
		expectedRuntime bool
	}{
		{
			name:            "Eager mode",
			mode:            Eager,
			expectedLazy:    false,
			expectedRuntime: false,
		},
		{
			name:            "Lazy mode",
			mode:            Lazy,
			expectedLazy:    true,
			expectedRuntime: true,
		},
		{
			name:            "EagerWithRuntime mode",
			mode:            EagerWithRuntime,
			expectedLazy:    false,
			expectedRuntime: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options{}
			opt := WithMode(tt.mode)
			opt(opts)

			if opts.lazy != tt.expectedLazy {
				t.Errorf("lazy: got %v, want %v", opts.lazy, tt.expectedLazy)
			}
			if opts.runtime != tt.expectedRuntime {
				t.Errorf("runtime: got %v, want %v", opts.runtime, tt.expectedRuntime)
			}
		})
	}

	t.Run("invalid mode should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("WithMode did not panic on invalid mode")
			}
		}()

		opts := &options{}
		opt := WithMode(Mode(999))
		opt(opts)
	})
}

func TestRepository_compile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		successCompiler := func(b []byte) (string, error) {
			return string(b), nil
		}

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
		errInternal := errors.New("internal error")
		failCompiler := func(b []byte) (string, error) {
			return "", errInternal
		}

		tests := []struct {
			name                string
			fs                  fstest.MapFS
			key                 string
			expectedError       error // 1段目のセンチネルエラー
			expectedInternalErr error // 2段目の内部エラー (任意)
		}{
			{
				name:          "filesystem error",
				fs:            fstest.MapFS{},
				key:           "missing.txt",
				expectedError: ErrNotFound,
				// expectedInternalErr は nil のまま
			},
			{
				name: "compiler error",
				fs: fstest.MapFS{
					"error.txt": {Data: []byte("some data")},
				},
				key:                 "error.txt",
				expectedError:       ErrCompile,
				expectedInternalErr: errInternal,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     tt.fs,
					compiler: failCompiler,
				}

				_, err := repo.compile(tt.key)
				if err == nil {
					t.Fatal("expected error but got nil")
				}

				if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error to wrap %v, got %v", tt.expectedError, err)
				}

				if tt.expectedInternalErr != nil {
					if !errors.Is(err, tt.expectedInternalErr) {
						t.Errorf("expected error to wrap internal error %v, got %v", tt.expectedInternalErr, err)
					}
				}
			})
		}
	})
}

// mockErrorFS は WalkDir 中に意図的にエラーを発生させるためのモックFSです
type mockErrorFS struct {
	fs.FS
}

func (m *mockErrorFS) Open(name string) (fs.File, error) {
	return nil, errors.New("filesystem read error")
}

func TestRepository_compileAll(t *testing.T) {
	mockFilter := func(path string) bool {
		return len(path) > 4 && strings.HasSuffix(path, ".txt")
	}

	t.Run("Success", func(t *testing.T) {
		successCompiler := func(b []byte) (string, error) {
			return string(b), nil
		}

		fsys := fstest.MapFS{
			"file1.txt":     {Data: []byte("content1")},
			"file2.txt":     {Data: []byte("content2")},
			"ignore.json":   {Data: []byte("content3")},
			"dir/file3.txt": {Data: []byte("content3")},
		}

		repo := &Repository[string]{
			fsys:     fsys,
			compiler: successCompiler,
			filter:   mockFilter,
		}

		err := repo.compileAll()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedFiles := []string{"file1.txt", "file2.txt", "dir/file3.txt"}
		for _, f := range expectedFiles {
			if _, ok := repo.resources.Load(f); !ok {
				t.Errorf("expected file %s to be compiled", f)
			}
		}

		if _, ok := repo.resources.Load("ignore.json"); ok {
			t.Error("expected ignore.json to be filtered out")
		}
	})

	t.Run("Failure", func(t *testing.T) {
		failCompiler := func(b []byte) (string, error) {
			return "", errors.New("mock compile error")
		}

		tests := []struct {
			name          string
			fsys          fs.FS
			expectedError error
		}{
			{
				name:          "walk error",
				fsys:          &mockErrorFS{},
				expectedError: ErrWalk,
			},
			{
				name: "compile error during all",
				fsys: fstest.MapFS{
					"bad.txt": {Data: []byte("some data")},
				},
				expectedError: ErrCompile,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &Repository[string]{
					fsys:     tt.fsys,
					compiler: failCompiler,
					filter:   mockFilter,
				}

				err := repo.compileAll()
				if err == nil {
					t.Fatal("expected error but got nil")
				}

				if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error to wrap %v, got %v", tt.expectedError, err)
				}
			})
		}
	})
}
