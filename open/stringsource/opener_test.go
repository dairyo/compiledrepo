package stringsource

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/dairyo/compiledrepo"
)

func TestOpener_Open(t *testing.T) {
	data := map[string]string{
		"key1": "content1",
		"key2": "content2",
	}
	opener := NewOpener(data)

	t.Run("SuccessCases", func(t *testing.T) {
		tests := []struct {
			name     string
			key      string
			wantRead string
		}{
			{
				name:     "valid key 1",
				key:      "key1",
				wantRead: "content1",
			},
			{
				name:     "valid key 2",
				key:      "key2",
				wantRead: "content2",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := context.Background()
				rc, err := opener.Open(ctx, tt.key)
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				defer rc.Close()

				content, err := io.ReadAll(rc)
				if err != nil {
					t.Fatalf("failed to read: %v", err)
				}
				if string(content) != tt.wantRead {
					t.Errorf("got %q, want %q", string(content), tt.wantRead)
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name    string
			key     string
			wantErr error
		}{
			{
				name:    "invalid key",
				key:     "invalid",
				wantErr: compiledrepo.ErrOpen,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := context.Background()
				_, err := opener.Open(ctx, tt.key)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got %v, want %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("ContextCases", func(t *testing.T) {
		tests := []struct {
			name    string
			wantErr error
		}{
			{
				name:    "cancelled context",
				wantErr: context.Canceled,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				_, err := opener.Open(ctx, "key1")
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got %v, want %v", err, tt.wantErr)
				}
			})
		}
	})
}
