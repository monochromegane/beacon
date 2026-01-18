package beacon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_Write(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	err := store.Write(12345, "test message")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "12345"))
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

	store.Write(12345, "first message")
	err := store.Write(12345, "second message")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "12345"))
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

	store.Write(12345, "test message")

	err := store.Delete(12345)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = os.Stat(filepath.Join(tmpDir, "12345"))
	if !os.IsNotExist(err) {
		t.Errorf("Delete() file still exists")
	}
}

func TestFileStore_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	err := store.Delete(99999)
	if err != nil {
		t.Errorf("Delete() error = %v, want nil for non-existent file", err)
	}
}

func TestFileStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	store.Write(12345, "message 1")
	store.Write(67890, "message 2")

	states, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("List() len = %d, want 2", len(states))
	}

	stateMap := make(map[int]string)
	for _, s := range states {
		stateMap[s.PID] = s.Message
	}

	if stateMap[12345] != "message 1" {
		t.Errorf("List() state[12345] = %q, want %q", stateMap[12345], "message 1")
	}
	if stateMap[67890] != "message 2" {
		t.Errorf("List() state[67890] = %q, want %q", stateMap[67890], "message 2")
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

func TestFileStore_List_IgnoresNonPIDFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStoreWithDir(tmpDir)

	store.Write(12345, "valid")
	os.WriteFile(filepath.Join(tmpDir, "not-a-pid"), []byte("invalid"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	states, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("List() len = %d, want 1", len(states))
	}
	if states[0].PID != 12345 {
		t.Errorf("List() state[0].PID = %d, want 12345", states[0].PID)
	}
}
