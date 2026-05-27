package fileloader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("hello world")
	filename := "test.txt"
	err := os.WriteFile(filepath.Join(tmpDir, filename), content, 0644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewFileLoader(tmpDir)
	got, err := loader.Load(context.Background(), filename)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("expected %q, got %q", string(content), string(got))
	}
}

func TestFileLoader_All(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{"a.txt", "b.txt", "dir/c.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	loader := NewFileLoader(tmpDir)
	ctx := context.Background()
	var found []string
	for id, err := range loader.All(ctx) {
		if err != nil {
			t.Fatalf("All yielded error: %v", err)
		}
		found = append(found, id)
	}

	if len(found) != len(files) {
		t.Errorf("expected %d files, found %d", len(files), len(found))
	}

	// Check if all files are present (order may vary)
	for _, expected := range files {
		foundExpected := false
		for _, f := range found {
			if f == filepath.ToSlash(expected) {
				foundExpected = true
				break
			}
		}
		if !foundExpected {
			t.Errorf("expected file %s not found in results %v", expected, found)
		}
	}
}

func TestFileLoader_All_ContextCancel(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 100; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune(i)) + ".txt")
		_ = os.WriteFile(filename, []byte("test"), 0644)
	}

	ctx, cancel := context.WithCancel(context.Background())
	loader := NewFileLoader(tmpDir)
	
	count := 0
	for _, err := range loader.All(ctx) {
		count++
		if count == 10 {
			cancel()
		}
		if err != nil {
			if err == context.Canceled {
				return // Success
			}
			t.Fatalf("unexpected error: %v", err)
		}
	}
	t.Error("expected context.Canceled error, but iteration finished normally")
}
