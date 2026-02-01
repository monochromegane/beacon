package signal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/monochromegane/beacon/internal/storage"
)

// Store is an interface for signal file operations.
type Store interface {
	Write(signal *Signal) error
	Delete(signalType, sessionID string) error
	Read(signalType, sessionID string) (*Signal, error)
	List(signalType string) ([]*Signal, error)
}

// FileStore is the file-based implementation of Store.
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

// filename returns the file name for a signal: {signal_type}_{session_id}.json
func filename(signalType, sessionID string) string {
	return signalType + "_" + sessionID + ".json"
}

// Write saves the signal as a JSON file.
func (s *FileStore) Write(signal *Signal) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(signal)
	if err != nil {
		return err
	}
	path := filepath.Join(s.baseDir, filename(signal.SignalType, signal.SessionID))
	return os.WriteFile(path, data, 0644)
}

// Delete removes the signal file for the given signal type and session ID.
// Returns nil if the file does not exist (idempotent).
func (s *FileStore) Delete(signalType, sessionID string) error {
	path := filepath.Join(s.baseDir, filename(signalType, sessionID))
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Read returns the signal for the given signal type and session ID.
func (s *FileStore) Read(signalType, sessionID string) (*Signal, error) {
	path := filepath.Join(s.baseDir, filename(signalType, sessionID))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var signal Signal
	if err := json.Unmarshal(data, &signal); err != nil {
		return nil, err
	}
	return &signal, nil
}

// List returns all signals of the given signal type.
func (s *FileStore) List(signalType string) ([]*Signal, error) {
	entries, err := os.ReadDir(s.baseDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	prefix := signalType + "_"
	suffix := ".json"
	var signals []*Signal
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.baseDir, name))
		if err != nil {
			continue
		}
		var signal Signal
		if err := json.Unmarshal(data, &signal); err != nil {
			continue
		}
		signals = append(signals, &signal)
	}
	return signals, nil
}
