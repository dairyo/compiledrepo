package bytesource

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/dairyo/compiledrepo"
)

func TestOpener_Open(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		store := map[string][]byte{
			"key1": []byte("content 1"),
			"key2": []byte("content 2"),
		}
		opener := NewOpener(store)
		ctx := context.Background()

		tests := []struct {
			name     string
			key      string
			wantData []byte
		}{
			{
				name:     "open existing key1",
				key:      "key1",
				wantData: []byte("content 1"),
			},
			{
				name:     "open existing key2",
				key:      "key2",
				wantData: []byte("content 2"),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reader, err := opener.Open(ctx, tt.key)
				if err != nil {
					t.Errorf("Open() error = %v, wantErr nil", err)
					return
				}

				data, err := io.ReadAll(reader)
				if err != nil {
					t.Fatalf("failed to read reader: %v", err)
				}

				if !bytes.Equal(data, tt.wantData) {
					t.Errorf("Open() data = %s, want %s", string(data), string(tt.wantData))
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		store := map[string][]byte{
			"key1": []byte("content 1"),
		}
		opener := NewOpener(store)
		ctx := context.Background()

		tests := []struct {
			name    string
			key     string
			wantErr error
		}{
			{
				name:    "key not found",
				key:     "non_existent_key",
				wantErr: compiledrepo.ErrOpen,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reader, err := opener.Open(ctx, tt.key)
				if reader != nil {
					t.Errorf("Open() returned reader but expected error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		store := map[string][]byte{
			"key1": []byte("content 1"),
		}
		opener := NewOpener(store)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		key := "key1"
		reader, err := opener.Open(ctx, key)

		if reader != nil {
			t.Errorf("expected nil reader, got %v", reader)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}
