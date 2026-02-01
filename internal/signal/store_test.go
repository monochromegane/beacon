package signal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStore_Write(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	sig := &Signal{
		SessionID:  "test123",
		SignalType: "claude",
		State:      StateRunning,
		Message:    "claude:running",
		UpdatedAt:  time.Now(),
	}

	err := store.Write(sig)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, "claude_test123.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Write() did not create file")
	}
}

func TestFileStore_Read(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	// Write a signal first
	sig := &Signal{
		SessionID:  "test123",
		SignalType: "claude",
		State:      StateRunning,
		Message:    "claude:running",
		UpdatedAt:  time.Now(),
	}
	if err := store.Write(sig); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read it back
	read, err := store.Read("claude", "test123")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if read.SessionID != "test123" {
		t.Errorf("SessionID = %q, want %q", read.SessionID, "test123")
	}
	if read.State != StateRunning {
		t.Errorf("State = %q, want %q", read.State, StateRunning)
	}
}

func TestFileStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	// Write a signal first
	sig := &Signal{
		SessionID:  "test123",
		SignalType: "claude",
		State:      StateRunning,
		Message:    "claude:running",
		UpdatedAt:  time.Now(),
	}
	if err := store.Write(sig); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Delete it
	err := store.Delete("claude", "test123")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file is gone
	path := filepath.Join(tmpDir, "claude_test123.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Delete() did not remove file")
	}
}

func TestFileStore_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	// Delete non-existent should not error
	err := store.Delete("claude", "nonexistent")
	if err != nil {
		t.Errorf("Delete() error = %v, want nil", err)
	}
}

func TestFileStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	// Write multiple signals
	signals := []*Signal{
		{SessionID: "test1", SignalType: "claude", State: StateRunning, Message: "msg1", UpdatedAt: time.Now()},
		{SessionID: "test2", SignalType: "claude", State: StateIdle, Message: "msg2", UpdatedAt: time.Now()},
		{SessionID: "test3", SignalType: "other", State: StateRunning, Message: "msg3", UpdatedAt: time.Now()},
	}
	for _, sig := range signals {
		if err := store.Write(sig); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	// List claude signals
	list, err := store.List("claude")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List() returned %d signals, want 2", len(list))
	}
}

func TestFileStore_List_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a non-existent subdirectory
	store := NewFileStoreWithDir(filepath.Join(tmpDir, "nonexistent"))

	list, err := store.List("claude")
	if err != nil {
		t.Errorf("List() error = %v, want nil", err)
	}
	if list != nil {
		t.Errorf("List() = %v, want nil", list)
	}
}
