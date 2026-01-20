package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileContextStore_Write(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileContextStoreWithDir(tmpDir)

	ctx := &TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
	}

	err := store.Write(12345, ctx)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "12345.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	expected := `{"session_name":"main","window_index":0,"pane_index":1,"pane_id":"%2"}`
	if string(content) != expected {
		t.Errorf("Write() content = %q, want %q", string(content), expected)
	}
}

func TestFileContextStore_Write_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileContextStoreWithDir(tmpDir)

	ctx1 := &TmuxContext{SessionName: "first", WindowIndex: 0, PaneIndex: 0, PaneID: "%0"}
	ctx2 := &TmuxContext{SessionName: "second", WindowIndex: 1, PaneIndex: 2, PaneID: "%3"}

	store.Write(12345, ctx1)
	err := store.Write(12345, ctx2)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "12345.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	expected := `{"session_name":"second","window_index":1,"pane_index":2,"pane_id":"%3"}`
	if string(content) != expected {
		t.Errorf("Write() content = %q, want %q", string(content), expected)
	}
}

func TestFileContextStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileContextStoreWithDir(tmpDir)

	ctx := &TmuxContext{SessionName: "main", WindowIndex: 0, PaneIndex: 0, PaneID: "%0"}
	store.Write(12345, ctx)

	err := store.Delete(12345)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = os.Stat(filepath.Join(tmpDir, "12345.json"))
	if !os.IsNotExist(err) {
		t.Errorf("Delete() file still exists")
	}
}

func TestFileContextStore_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileContextStoreWithDir(tmpDir)

	err := store.Delete(99999)
	if err != nil {
		t.Errorf("Delete() error = %v, want nil for non-existent file", err)
	}
}

func TestFileContextStore_Read(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileContextStoreWithDir(tmpDir)

	ctx := &TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
	}
	store.Write(12345, ctx)

	data, err := store.Read(12345)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	expected := `{"session_name":"main","window_index":0,"pane_index":1,"pane_id":"%2"}`
	if string(data) != expected {
		t.Errorf("Read() = %q, want %q", string(data), expected)
	}
}

func TestFileContextStore_Read_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileContextStoreWithDir(tmpDir)

	_, err := store.Read(99999)
	if err == nil {
		t.Error("Read() expected error for non-existent file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Read() error = %v, want os.IsNotExist error", err)
	}
}
