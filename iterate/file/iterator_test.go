package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dairyo/compiledrepo"
)

func TestIterator_All(t *testing.T) {
	t.Run("SuccessCases", func(t *testing.T) {
		tmpDir := t.TempDir()
		files := []string{"file1.txt", "file2.txt", "file3.txt"}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("content"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		// Create a directory to ensure it's skipped
		if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
			t.Fatal(err)
		}

		it := NewIterator(tmpDir)
		ctx := context.Background()

		var got []string
		for key, err := range it.All(ctx) {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			got = append(got, key)
		}

		if len(got) != len(files) {
			t.Errorf("expected %d files, got %d", len(files), len(got))
		}
		for _, f := range files {
			found := false
			for _, g := range got {
				if f == g {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("file %s not found in result", f)
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name    string
			dirPath string
			wantErr error
		}{
			{
				name:    "NonExistentDirectory",
				dirPath: "non_existent_dir_12345",
				wantErr: compiledrepo.ErrIterator,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				it := NewIterator(tt.dirPath)
				ctx := context.Background()

				var err error
				for key, e := range it.All(ctx) {
					if e != nil {
						err = e
						break
					}
					_ = key
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			})
		}
	})

	t.Run("CancellationCases", func(t *testing.T) {
		tmpDir := t.TempDir()
		files := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt"}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("content"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		it := NewIterator(tmpDir)

		var count int
		for key, err := range it.All(ctx) {
			count++
			cancel() // cancel immediately after first element
			if err != nil {
				break
			}
			_ = key
		}

		// It should have yielded at least one element and then stop due to cancellation.
		// Depending on timing, it might yield more than one if we cancel within the loop.
		if count == 0 {
			t.Error("expected at least one element before cancellation")
		}
	})
}
