package compiledrepo

import (
	"context"
	"errors"
	"testing"
)

// MockLoader is a simple mock implementation of the Loader interface.
type MockLoader struct {
	data map[string][]byte
}

func (m *MockLoader) Load(ctx context.Context, id string) ([]byte, error) {
	data, ok := m.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	return data, nil
}

// MockPreloader is a simple mock implementation of the Preloader interface.
type MockPreloader struct {
	ids []string
}

func (m *MockPreloader) All(ctx context.Context) func(func(string, error) bool) {
	return func(yield func(string, error) bool) {
		for _, id := range m.ids {
			if !yield(id, nil) {
				return
			}
		}
	}
}

func TestMockLoader_Load(t *testing.T) {
	loader := &MockLoader{
		data: map[string][]byte{"test": []byte("value")},
	}
	ctx := context.Background()

	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name    string
			id      string
			wantVal string
		}{
			{"LoadExisting", "test", "value"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				val, err := loader.Load(ctx, tt.id)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if string(val) != tt.wantVal {
					t.Errorf("expected %s, got %s", tt.wantVal, string(val))
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name    string
			id      string
			wantErr error
		}{
			{"LoadMissing", "missing", ErrNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := loader.Load(ctx, tt.id)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got %v", tt.wantErr, err)
				}
			})
		}
	})
}

func TestCompilerType(t *testing.T) {
	compiler := func(b []byte) (int, error) {
		if len(b) == 0 {
			return 0, errors.New("empty data")
		}
		return len(b), nil
	}

	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name  string
			input []byte
			want  int
		}{
			{"ValidInput", []byte("hello"), 5},
			{"SingleChar", []byte("a"), 1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				val, err := compiler(tt.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if val != tt.want {
					t.Errorf("expected %d, got %d", tt.want, val)
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name  string
			input []byte
		}{
			{"EmptyInput", []byte{}},
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
