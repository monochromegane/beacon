package beacon

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// State represents the content of a beacon state file.
type State struct {
	PID     int
	Message string
}

// Store is an interface for file operations (mockable for tests).
type Store interface {
	Write(pid int, message string) error
	Delete(pid int) error
	List() ([]State, error)
}

// FileStore is the production implementation of Store.
type FileStore struct {
	baseDir string
}

// NewFileStore creates a new FileStore with the resolved base directory.
func NewFileStore() (*FileStore, error) {
	baseDir, err := resolveBaseDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{baseDir: baseDir}, nil
}

// NewFileStoreWithDir creates a new FileStore with a custom base directory (for testing).
func NewFileStoreWithDir(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

// Write creates or updates a state file for the given PID.
func (s *FileStore) Write(pid int, message string) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(s.baseDir, strconv.Itoa(pid))
	return os.WriteFile(path, []byte(message), 0644)
}

// Delete removes the state file for the given PID.
// Returns nil if the file does not exist (idempotent).
func (s *FileStore) Delete(pid int) error {
	path := filepath.Join(s.baseDir, strconv.Itoa(pid))
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// List returns all states from beacon files in the base directory.
// Returns an empty slice if the directory does not exist.
func (s *FileStore) List() ([]State, error) {
	entries, err := os.ReadDir(s.baseDir)
	if os.IsNotExist(err) {
		return []State{}, nil
	}
	if err != nil {
		return nil, err
	}

	var states []State
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		path := filepath.Join(s.baseDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		states = append(states, State{
			PID:     pid,
			Message: strings.TrimSpace(string(content)),
		})
	}
	return states, nil
}

// resolveBaseDir returns the base directory for beacon state files.
func resolveBaseDir() (string, error) {
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "beacon"), nil
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCache, "beacon"), nil
}
