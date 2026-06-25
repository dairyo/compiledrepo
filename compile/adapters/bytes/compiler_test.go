package bytes

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/dairyo/compiledrepo"
)

// mockReadCloser allows simulating read errors and tracking Close().
type mockReadCloser struct {
	*bytes.Reader
	closed bool
	fail   bool
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.fail {
		return 0, errors.New("simulated read error")
	}
	return m.Reader.Read(p)
}

func TestCompiler_Compile(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		t.Run("ByteSlice", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
				want  []byte
			}{
				{"empty", "", []byte("")},
				{"basic", "hello", []byte("hello")},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					c := NewCompiler[[]byte]()
					r := &mockReadCloser{Reader: bytes.NewReader([]byte(tt.input))}
					got, err := c.Compile(context.Background(), r)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if !bytes.Equal(got, tt.want) {
						t.Errorf("got %v, want %v", got, tt.want)
					}
					if !r.closed {
						t.Error("expected reader to be closed")
					}
				})
			}
		})

		t.Run("String", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
				want  string
			}{
				{"empty", "", ""},
				{"basic", "hello", "hello"},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					c := NewCompiler[string]()
					r := &mockReadCloser{Reader: bytes.NewReader([]byte(tt.input))}
					got, err := c.Compile(context.Background(), r)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if got != tt.want {
						t.Errorf("got %q, want %q", got, tt.want)
					}
					if !r.closed {
						t.Error("expected reader to be closed")
					}
				})
			}
		})
	})

	t.Run("ErrorCases", func(t *testing.T) {
		t.Run("ReadError", func(t *testing.T) {
			c := NewCompiler[string]()
			r := &mockReadCloser{
				Reader: bytes.NewReader([]byte("foo")),
				fail:   true,
			}
			_, err := c.Compile(context.Background(), r)
			if !errors.Is(err, compiledrepo.ErrCompile) {
				t.Errorf("expected ErrCompile, got %v", err)
			}
			if !r.closed {
				t.Error("expected reader to be closed even on error")
			}
		})

		t.Run("NilReader", func(t *testing.T) {
			c := NewCompiler[string]()
			_, err := c.Compile(context.Background(), nil)
			if !errors.Is(err, compiledrepo.ErrCompile) {
				t.Errorf("expected ErrCompile, got %v", err)
			}
		})

		t.Run("UnsupportedType", func(t *testing.T) {
			c := NewCompiler[int]()
			r := &mockReadCloser{Reader: bytes.NewReader([]byte("123"))}
			_, err := c.Compile(context.Background(), r)
			if !errors.Is(err, compiledrepo.ErrCompile) {
				t.Errorf("expected ErrCompile, got %v", err)
			}
			if !r.closed {
				t.Error("expected reader to be closed")
			}
		})

		t.Run("ContextCancelled", func(t *testing.T) {
			c := NewCompiler[string]()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			r := &mockReadCloser{Reader: bytes.NewReader([]byte("foo"))}
			_, err := c.Compile(ctx, r)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled, got %v", err)
			}
			if !r.closed {
				t.Error("expected reader to be closed")
			}
		})
	})
}
