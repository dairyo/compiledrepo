package compiledrepo

import (
	"testing"
)

func TestRegistry_Get(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name     string
			key      string
			expected string
		}{
			{
				name:     "RetrieveExistingKey",
				key:      "key1",
				expected: "value1",
			},
			{
				name:     "RetrieveAnotherExistingKey",
				key:      "key2",
				expected: "value2",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reg := NewRegistry[string, string]()
				reg.Set("key1", "value1")
				reg.Set("key2", "value2")

				val, ok := reg.Get(tt.key)
				if !ok {
					t.Errorf("Get(%s) ok = false, want true", tt.key)
				}
				if val != tt.expected {
					t.Errorf("Get(%s) val = %v, want %v", tt.key, val, tt.expected)
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name string
			key  string
		}{
			{
				name: "RetrieveNonExistentKey",
				key:  "nonexistent",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reg := NewRegistry[string, string]()
				reg.Set("existing", "value")

				val, ok := reg.Get(tt.key)
				if ok {
					t.Errorf("Get(%s) ok = true, want false", tt.key)
				}
				if val != "" {
					t.Errorf("Get(%s) val = %v, want empty string", tt.key, val)
				}
			})
		}
	})

	t.Run("NilRegistryCases", func(t *testing.T) {
		tests := []struct {
			name string
			key  string
		}{
			{
				name: "NilRegistryReturnsFalse",
				key:  "anykey",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var reg *Registry[string, string]
				val, ok := reg.Get(tt.key)
				if ok {
					t.Errorf("Get(%s) on nil registry ok = true, want false", tt.key)
				}
				if val != "" {
					t.Errorf("Get(%s) on nil registry val = %v, want empty string", tt.key, val)
				}
			})
		}
	})
}
