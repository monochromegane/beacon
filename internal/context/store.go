package context

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/monochromegane/beacon/internal/storage"
)

// ContextStore handles persistence of context information as JSON files.
type ContextStore interface {
	Write(pid int, ctx Context) error
	Delete(pid int) error
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

// Write saves the context as a JSON file for the given PID.
func (s *FileContextStore) Write(pid int, ctx Context) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	data, err := ctx.ToJSON()
	if err != nil {
		return err
	}
	path := filepath.Join(s.baseDir, strconv.Itoa(pid)+".json")
	return os.WriteFile(path, data, 0644)
}

// Delete removes the context JSON file for the given PID.
// Returns nil if the file does not exist (idempotent).
func (s *FileContextStore) Delete(pid int) error {
	path := filepath.Join(s.baseDir, strconv.Itoa(pid)+".json")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
