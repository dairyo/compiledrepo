package compiledrepo

import (
	"errors"
	"testing"
)

func TestRegistry_Get(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name string
			id   string
			want string
		}{
			{
				name: "retrieve existing resource",
				id:   "res1",
				want: "value1",
			},
			{
				name: "retrieve another existing resource",
				id:   "res2",
				want: "value2",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reg := &Registry[string]{
					resources: map[string]string{
						"res1": "value1",
						"res2": "value2",
					},
				}
				got, err := reg.Get(tt.id)
				if err != nil {
					t.Fatalf("Get() error = %v, wantErr nil", err)
				}
				if got != tt.want {
					t.Errorf("Get() got = %v, want %v", got, tt.want)
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
			{
				name:    "resource not found",
				id:      "nonexistent",
				wantErr: ErrNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reg := &Registry[string]{
					resources: map[string]string{
						"res1": "value1",
					},
				}
				got, err := reg.Get(tt.id)
				if err != nil {
					if !errors.Is(err, tt.wantErr) {
						t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
					}
				} else {
					t.Errorf("Get() error = nil, wantErr %v", tt.wantErr)
				}
				// Ensure zero value is returned on error
				var zero string
				if got != zero {
					t.Errorf("Get() got = %v, want zero value %v", got, zero)
				}
			})
		}
	})
}
