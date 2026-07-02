package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dairyo/compiledrepo"
)

func TestOpener_Open(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		// Create a temporary file for testing
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test_file.txt")
		if err := os.WriteFile(tmpFile, []byte("hello world"), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		opener := NewOpener()
		ctx := context.Background()

		tests := []struct {
			name string
			path string
			want string
		}{
			{
				name: "open existing file",
				path: tmpFile,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f, err := opener.Open(ctx, tt.path)
				if err != nil {
					t.Errorf("Open() error = %v, wantErr nil", err)
					return
				}
				defer f.Close()

				// Verify the file is opened
				if f == nil {
					t.Error("expected file handle, got nil")
				}
			})
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		opener := NewOpener()
		ctx := context.Background()

		tests := []struct {
			name    string
			path    string
			wantErr error
		}{
			{
				name:    "file not found",
				path:    "non_existent_file_xyz_123.txt",
				wantErr: compiledrepo.ErrOpen,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f, err := opener.Open(ctx, tt.path)
				if f != nil {
					f.Close()
					t.Errorf("Open() returned file handle but expected error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})
}
