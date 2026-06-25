package string

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/dairyo/compiledrepo"
)

// mockReadCloser implements io.ReadCloser for testing.
type mockReadCloser struct {
	data []byte
	err  error
	read int
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.read >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.read:])
	m.read += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}

func TestCompiler_Compile(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		t.Run("ToString", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
				want  string
			}{
				{"Simple", "hello", "hello"},
				{"Empty", "", ""},
				{"Long", "a very long string", "a very long string"},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					c := NewCompiler[string]()
					r := &mockReadCloser{data: []byte(tt.input)}
					got, err := c.Compile(context.Background(), r)
					if err != nil {
						t.Fatalf("Compile() unexpected error: %v", err)
					}
					if got != tt.want {
						t.Errorf("Compile() got = %v, want %v", got, tt.want)
					}
				})
			}
		})

		t.Run("ToBytes", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
				want  []byte
			}{
				{"Simple", "hello", []byte("hello")},
				{"Empty", "", []byte("")},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					c := NewCompiler[[]byte]()
					r := &mockReadCloser{data: []byte(tt.input)}
					got, err := c.Compile(context.Background(), r)
					if err != nil {
						t.Fatalf("Compile() unexpected error: %v", err)
					}
					if !reflect.DeepEqual(got, tt.want) {
						t.Errorf("Compile() got = %v, want %v", got, tt.want)
					}
				})
			}
		})
	})

	t.Run("ErrorCases", func(t *testing.T) {
		t.Run("ReadError", func(t *testing.T) {
			tests := []struct {
				name    string
				readErr error
			}{
				{"GenericError", errors.New("read failed")},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					c := NewCompiler[string]()
					r := &mockReadCloser{err: tt.readErr}
					_, err := c.Compile(context.Background(), r)
					if !errors.Is(err, compiledrepo.ErrCompile) {
						t.Errorf("Compile() error = %v, want it to wrap %v", err, compiledrepo.ErrCompile)
					}
				})
			}
		})

		t.Run("NilReader", func(t *testing.T) {
			c := NewCompiler[string]()
			_, err := c.Compile(context.Background(), nil)
			if !errors.Is(err, compiledrepo.ErrCompile) {
				t.Errorf("Compile() error = %v, want it to wrap %v", err, compiledrepo.ErrCompile)
			}
		})

		t.Run("UnsupportedType", func(t *testing.T) {
			c := NewCompiler[int]()
			r := &mockReadCloser{data: []byte("123")}
			_, err := c.Compile(context.Background(), r)
			if !errors.Is(err, compiledrepo.ErrCompile) {
				t.Errorf("Compile() error = %v, want it to wrap %v", err, compiledrepo.ErrCompile)
			}
		})

		t.Run("ContextCancelled", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			c := NewCompiler[string]()
			r := &mockReadCloser{data: []byte("hello")}
			_, err := c.Compile(ctx, r)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Compile() error = %v, want %v", err, context.Canceled)
			}
		})
	})
}
