package context

import (
	"os"
	"path/filepath"

	"github.com/monochromegane/beacon/internal/storage"
)

// ContextStore handles persistence of context information as JSON files.
type ContextStore interface {
	Write(id string, ctx Context) error
	Delete(id string) error
	Read(id string) ([]byte, error)
}

// FileContextStore is the file-based implementation of ContextStore.
type FileContextStore struct {
	baseDir string
}

// NewFileContextStore creates a new FileContextStore with the resolved base directory.
func NewFileContextStore() (*FileContextStore, error) {
	baseDir, err := storage.ResolveBaseDir()
	if err != nil {
		return nil, err
	}
	return &FileContextStore{baseDir: baseDir}, nil
}

// NewFileContextStoreWithDir creates a new FileContextStore with a custom base directory (for testing).
func NewFileContextStoreWithDir(baseDir string) *FileContextStore {
	return &FileContextStore{baseDir: baseDir}
}

// Write saves the context as a JSON file for the given ID.
func (s *FileContextStore) Write(id string, ctx Context) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	data, err := ctx.ToJSON()
	if err != nil {
		return err
	}
	path := filepath.Join(s.baseDir, id+".json")
	return os.WriteFile(path, data, 0644)
}

// Delete removes the context JSON file for the given ID.
// Returns nil if the file does not exist (idempotent).
func (s *FileContextStore) Delete(id string) error {
	path := filepath.Join(s.baseDir, id+".json")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Read returns the raw JSON content of the context file for the given ID.
func (s *FileContextStore) Read(id string) ([]byte, error) {
	path := filepath.Join(s.baseDir, id+".json")
	return os.ReadFile(path)
}
