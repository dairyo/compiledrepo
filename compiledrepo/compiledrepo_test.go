package compiledrepo

import (
	"context"
	"errors"
	"iter"
	"testing"
)

// MockLoader implements Loader for testing.
type MockLoader struct {
	data map[string][]byte
}

func (m *MockLoader) Load(ctx context.Context, id string) ([]byte, error) {
	if data, ok := m.data[id]; ok {
		return data, nil
	}
	return nil, errors.New("not found")
}

// MockPreloader implements Preloader for testing.
type MockPreloader struct {
	ids    []string
	errSeq error
}

func (m *MockPreloader) All(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		if m.errSeq != nil {
			if !yield("", m.errSeq) {
				return
			}
			return
		}
		for _, id := range m.ids {
			if !yield(id, nil) {
				return
			}
		}
	}
}

func TestLoader(t *testing.T) {
	var _ Loader = (*MockLoader)(nil)

	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name string
			id   string
			data map[string][]byte
			want []byte
		}{
			{
				name: "LoadExisting",
				id:   "test",
				data: map[string][]byte{"test": []byte("hello")},
				want: []byte("hello"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				loader := &MockLoader{data: tt.data}
				res, err := loader.Load(context.Background(), tt.id)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if string(res) != string(tt.want) {
					t.Errorf("got %s, want %s", string(res), string(tt.want))
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name string
			id   string
			data map[string][]byte
		}{
			{
				name: "LoadNonExisting",
				id:   "missing",
				data: map[string][]byte{"test": []byte("hello")},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				loader := &MockLoader{data: tt.data}
				_, err := loader.Load(context.Background(), tt.id)
				if err == nil {
					t.Error("expected error, got nil")
				}
			})
		}
	})
}

func TestCompiler(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		var compiler Compiler[string] = func(b []byte) (string, error) {
			return string(b), nil
		}

		tests := []struct {
			name  string
			input []byte
			want  string
		}{
			{
				name:  "CompileSimple",
				input: []byte("hello"),
				want:  "hello",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				res, err := compiler(tt.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res != tt.want {
					t.Errorf("got %s, want %s", res, tt.want)
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		var compiler Compiler[string] = func(b []byte) (string, error) {
			return "", errors.New("compile error")
		}

		tests := []struct {
			name  string
			input []byte
		}{
			{
				name:  "CompileFail",
				input: []byte("bad"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := compiler(tt.input)
				if err == nil {
					t.Error("expected error, got nil")
				}
			})
		}
	})
}

func TestPreloader(t *testing.T) {
	var _ Preloader = (*MockPreloader)(nil)

	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name string
			ids  []string
			want int
		}{
			{
				name: "PreloadMultiple",
				ids:  []string{"id1", "id2"},
				want: 2,
			},
			{
				name: "PreloadEmpty",
				ids:  []string{},
				want: 0,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				preloader := &MockPreloader{ids: tt.ids}
				count := 0
				for id, err := range preloader.All(context.Background()) {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if id == "" {
						t.Error("returned empty id")
					}
					count++
				}
				if count != tt.want {
					t.Errorf("got %d items, want %d", count, tt.want)
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
		}{
			{
				name: "PreloadError",
				err:  errors.New("preload error"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				preloader := &MockPreloader{errSeq: tt.err}
				errFound := false
				for _, err := range preloader.All(context.Background()) {
					if err != nil {
						errFound = true
						break
					}
				}
				if !errFound {
					t.Error("expected error, got nil")
				}
			})
		}
	})
}
