package fileloader

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"iter"
)

type FileLoader struct {
	rootDir string
}

func NewFileLoader(rootDir string) *FileLoader {
	return &FileLoader{rootDir: rootDir}
}

func (f *FileLoader) Load(ctx context.Context, id string) ([]byte, error) {
	// id is treated as a relative path from rootDir
	return os.ReadFile(filepath.Join(f.rootDir, id))
}

// stopWalkError is used to stop filepath.WalkDir without returning a real error to the yield function.
type stopWalkError struct{}

func (e stopWalkError) Error() string { return "stop walking" }

func (f *FileLoader) All(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		err := filepath.WalkDir(f.rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			// Check for context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Calculate relative path as the identifier
			rel, err := filepath.Rel(f.rootDir, path)
			if err != nil {
				return err
			}
			
			// Normalize path to use forward slashes for consistency across OS
			rel = filepath.ToSlash(rel)

			if !yield(rel, nil) {
				return stopWalkError{}
			}
			return nil
		})
		if err != nil && err != (stopWalkError{}) {
			yield("", err)
		}
	}
}
