package beacon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_Write(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	err := store.Write("test123", "test message")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "test123"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "test message" {
		t.Errorf("Write() content = %q, want %q", string(content), "test message")
	}
}

func TestFileStore_Write_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	store.Write("test123", "first message")
	err := store.Write("test123", "second message")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "test123"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "second message" {
		t.Errorf("Write() content = %q, want %q", string(content), "second message")
	}
}

func TestFileStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	store.Write("test123", "test message")

	err := store.Delete("test123")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = os.Stat(filepath.Join(tmpDir, "test123"))
	if !os.IsNotExist(err) {
		t.Errorf("Delete() file still exists")
	}
}

func TestFileStore_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	err := store.Delete("nonexistent")
	if err != nil {
		t.Errorf("Delete() error = %v, want nil for non-existent file", err)
	}
}

func TestFileStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	store.Write("session-abc", "message 1")
	store.Write("session-xyz", "message 2")

	states, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("List() len = %d, want 2", len(states))
	}

	stateMap := make(map[string]string)
	for _, s := range states {
		stateMap[s.ID] = s.Message
	}

	if stateMap["session-abc"] != "message 1" {
		t.Errorf("List() state[session-abc] = %q, want %q", stateMap["session-abc"], "message 1")
	}
	if stateMap["session-xyz"] != "message 2" {
		t.Errorf("List() state[session-xyz] = %q, want %q", stateMap["session-xyz"], "message 2")
	}
}

func TestFileStore_List_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	states, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(states) != 0 {
		t.Errorf("List() len = %d, want 0", len(states))
	}
}

func TestFileStore_List_NonExistentDir(t *testing.T) {
	store := NewFileStoreWithDir("/nonexistent/path")

	states, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(states) != 0 {
		t.Errorf("List() len = %d, want 0", len(states))
	}
}

func TestFileStore_List_IgnoresJSONFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	store.Write("test123", "valid")
	os.WriteFile(filepath.Join(tmpDir, "test123.json"), []byte("context"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	states, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("List() len = %d, want 1", len(states))
	}
	if states[0].ID != "test123" {
		t.Errorf("List() state[0].ID = %s, want test123", states[0].ID)
	}
}
