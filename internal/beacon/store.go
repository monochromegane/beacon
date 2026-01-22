package beacon

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/monochromegane/beacon/internal/storage"
)

// State represents the content of a beacon state file.
type State struct {
	ID      string
	Message string
}

// Store is an interface for file operations (mockable for tests).
type Store interface {
	Write(id string, message string) error
	Delete(id string) error
	List() ([]State, error)
}

// FileStore is the production implementation of Store.
type FileStore struct {
	baseDir string
}

// NewFileStore creates a new FileStore with the resolved base directory.
func NewFileStore() (*FileStore, error) {
	baseDir, err := storage.ResolveBaseDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{baseDir: baseDir}, nil
}

// NewFileStoreWithDir creates a new FileStore with a custom base directory (for testing).
func NewFileStoreWithDir(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

// Write creates or updates a state file for the given ID.
func (s *FileStore) Write(id string, message string) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(s.baseDir, id)
	return os.WriteFile(path, []byte(message), 0644)
}

// Delete removes the state file for the given ID.
// Returns nil if the file does not exist (idempotent).
func (s *FileStore) Delete(id string) error {
	path := filepath.Join(s.baseDir, id)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// List returns all states from beacon files in the base directory.
// Returns nil if the directory does not exist.
func (s *FileStore) List() ([]State, error) {
	entries, err := os.ReadDir(s.baseDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var states []State
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip JSON context files
		if strings.HasSuffix(name, ".json") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(s.baseDir, name))
		if err != nil {
			continue
		}
		states = append(states, State{
			ID:      name,
			Message: strings.TrimSpace(string(content)),
		})
	}
	return states, nil
}
