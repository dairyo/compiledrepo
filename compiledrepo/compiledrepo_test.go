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
	ids []string
}

func (m *MockPreloader) All(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for _, id := range m.ids {
			if !yield(id, nil) {
				return
			}
		}
	}
}

func TestInterfaces(t *testing.T) {
	t.Run("LoaderCompatibility", func(t *testing.T) {
		var _ Loader = (*MockLoader)(nil)

		loader := &MockLoader{data: map[string][]byte{"test": []byte("hello")}}
		ctx := context.Background()
		res, err := loader.Load(ctx, "test")
		if err != nil || string(res) != "hello" {
			t.Errorf("Loader.Load failed: got %s, %v; want hello, nil", string(res), err)
		}
	})

	t.Run("CompilerCompatibility", func(t *testing.T) {
		var compiler Compiler[string] = func(b []byte) (string, error) {
			return string(b), nil
		}

		res, err := compiler([]byte("hello"))
		if err != nil || res != "hello" {
			t.Errorf("Compiler failed: got %s, %v; want hello, nil", res, err)
		}
	})

	t.Run("PreloaderCompatibility", func(t *testing.T) {
		var _ Preloader = (*MockPreloader)(nil)

		preloader := &MockPreloader{ids: []string{"id1", "id2"}}
		ctx := context.Background()

		count := 0
		for id, err := range preloader.All(ctx) {
			if err != nil {
				t.Errorf("Preloader.All returned error: %v", err)
			}
			if id == "" {
				t.Error("Preloader.All returned empty id")
			}
			count++
		}
		if count != 2 {
			t.Errorf("Preloader.All returned %d items; want 2", count)
		}
	})
}
