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

func TestInterfaces(t *testing.T) {
	t.Run("LoaderInterface", func(t *testing.T) {
		loader := &MockLoader{
			data: map[string][]byte{"test": []byte("value")},
		}
		ctx := context.Background()

		t.Run("Success", func(t *testing.T) {
			val, err := loader.Load(ctx, "test")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if string(val) != "value" {
				t.Errorf("expected 'value', got %s", string(val))
			}
		})

		t.Run("NotFound", func(t *testing.T) {
			_, err := loader.Load(ctx, "missing")
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("CompilerType", func(t *testing.T) {
		compiler := func(b []byte) (int, error) {
			if len(b) == 0 {
				return 0, errors.New("empty data")
			}
			return len(b), nil
		}

		t.Run("Success", func(t *testing.T) {
			val, err := compiler([]byte("hello"))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if val != 5 {
				t.Errorf("expected 5, got %d", val)
			}
		})

		t.Run("Error", func(t *testing.T) {
			_, err := compiler([]byte{})
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	})
}
