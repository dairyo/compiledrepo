package compiledrepo

import (
	"context"
	"fmt"
	"testing"
)

// MockLoader implements the Loader interface for testing.
type MockLoader struct {
	data map[string][]byte
}

func (m *MockLoader) Load(ctx context.Context, id string) ([]byte, error) {
	if data, ok := m.data[id]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("not found")
}

// MockCompiler converts bytes to string for testing.
func MockCompiler(b []byte) (string, error) {
	return string(b), nil
}

func TestCoreDefinitions(t *testing.T) {
	t.Run("InterfaceAssignment", func(t *testing.T) {
		// Verify that MockLoader can be assigned to Loader interface.
		var loader Loader = &MockLoader{
			data: map[string][]byte{
				"test": []byte("hello"),
			},
		}
		if loader == nil {
			t.Fatal("Expected loader to be non-nil")
		}

		// Verify that MockCompiler can be assigned to Compiler[string] type.
		var compiler Compiler[string] = MockCompiler
		if compiler == nil {
			t.Fatal("Expected compiler to be non-nil")
		}

		// Verify that Repository can be instantiated with these types.
		repo := Repository[string]{
			loader:   loader,
			compiler: compiler,
		}
		if repo.loader != loader {
			t.Error("Repository loader was not correctly assigned")
		}
	})
}
