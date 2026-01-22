package beacon

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/monochromegane/beacon/internal/context"
)

// mockStore is a mock implementation of Store for testing.
type mockStore struct {
	states   map[string]string
	writeErr error
	delErr   error
	listErr  error
}

func newMockStore() *mockStore {
	return &mockStore{
		states: make(map[string]string),
	}
}

func (m *mockStore) Write(id string, message string) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.states[id] = message
	return nil
}

func (m *mockStore) Delete(id string) error {
	if m.delErr != nil {
		return m.delErr
	}
	delete(m.states, id)
	return nil
}

func (m *mockStore) List() ([]State, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var states []State
	for id, msg := range m.states {
		states = append(states, State{ID: id, Message: msg})
	}
	return states, nil
}

// mockContextStore is a mock implementation of context.ContextStore for testing.
type mockContextStore struct {
	contexts map[string]context.Context
	writeErr error
	delErr   error
}

func newMockContextStore() *mockContextStore {
	return &mockContextStore{
		contexts: make(map[string]context.Context),
	}
}

func (m *mockContextStore) Write(id string, ctx context.Context) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.contexts[id] = ctx
	return nil
}

func (m *mockContextStore) Delete(id string) error {
	if m.delErr != nil {
		return m.delErr
	}
	delete(m.contexts, id)
	return nil
}

func (m *mockContextStore) Read(id string) ([]byte, error) {
	ctx, ok := m.contexts[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return ctx.ToJSON()
}

func TestBeacon_Emit(t *testing.T) {
	store := newMockStore()
	b := New(store, nil)

	err := b.Emit("test123", "test message")
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if store.states["test123"] != "test message" {
		t.Errorf("Emit() message = %q, want %q", store.states["test123"], "test message")
	}
}

func TestBeacon_Emit_Error(t *testing.T) {
	store := newMockStore()
	store.writeErr = errors.New("write error")
	b := New(store, nil)

	err := b.Emit("test123", "test message")
	if err == nil {
		t.Error("Emit() expected error, got nil")
	}
}

func TestBeacon_Silence(t *testing.T) {
	store := newMockStore()
	store.states["test123"] = "test message"
	b := New(store, nil)

	err := b.Silence("test123")
	if err != nil {
		t.Fatalf("Silence() error = %v", err)
	}

	if _, exists := store.states["test123"]; exists {
		t.Error("Silence() state still exists")
	}
}

func TestBeacon_Silence_Error(t *testing.T) {
	store := newMockStore()
	store.delErr = errors.New("delete error")
	b := New(store, nil)

	err := b.Silence("test123")
	if err == nil {
		t.Error("Silence() expected error, got nil")
	}
}

func TestBeacon_List(t *testing.T) {
	store := newMockStore()
	store.states["test123"] = "message 1"
	var buf bytes.Buffer
	b := New(store, &buf)

	err := b.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	output := buf.String()
	expected := "test123\tmessage 1\n"
	if output != expected {
		t.Errorf("List() output = %q, want %q", output, expected)
	}
}

func TestBeacon_List_Empty(t *testing.T) {
	store := newMockStore()
	var buf bytes.Buffer
	b := New(store, &buf)

	err := b.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if buf.String() != "" {
		t.Errorf("List() output = %q, want empty", buf.String())
	}
}

func TestBeacon_List_Error(t *testing.T) {
	store := newMockStore()
	store.listErr = errors.New("list error")
	var buf bytes.Buffer
	b := New(store, &buf)

	err := b.List()
	if err == nil {
		t.Error("List() expected error, got nil")
	}
}

// mockContext implements context.Context for testing.
type mockContext struct {
	contextType string
	json        []byte
	jsonErr     error
}

func (m *mockContext) Type() string {
	return m.contextType
}

func (m *mockContext) ToJSON() ([]byte, error) {
	return m.json, m.jsonErr
}

func TestBeacon_EmitWithContext(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	b := NewWithContextStore(store, contextStore, nil)

	ctx := &mockContext{
		contextType: "tmux",
		json:        []byte(`{"session_name":"main"}`),
	}

	err := b.EmitWithContext("test123", "test message", ctx)
	if err != nil {
		t.Fatalf("EmitWithContext() error = %v", err)
	}

	if store.states["test123"] != "test message" {
		t.Errorf("EmitWithContext() message = %q, want %q", store.states["test123"], "test message")
	}

	if contextStore.contexts["test123"] == nil {
		t.Error("EmitWithContext() context not saved")
	}
}

func TestBeacon_EmitWithContext_StoreError(t *testing.T) {
	store := newMockStore()
	store.writeErr = errors.New("write error")
	contextStore := newMockContextStore()
	b := NewWithContextStore(store, contextStore, nil)

	ctx := &mockContext{contextType: "tmux", json: []byte(`{}`)}

	err := b.EmitWithContext("test123", "test message", ctx)
	if err == nil {
		t.Error("EmitWithContext() expected error, got nil")
	}
}

func TestBeacon_EmitWithContext_ContextStoreError(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	contextStore.writeErr = errors.New("context write error")
	b := NewWithContextStore(store, contextStore, nil)

	ctx := &mockContext{contextType: "tmux", json: []byte(`{}`)}

	err := b.EmitWithContext("test123", "test message", ctx)
	if err == nil {
		t.Error("EmitWithContext() expected error, got nil")
	}
}

func TestBeacon_EmitWithContext_NilContextStore(t *testing.T) {
	store := newMockStore()
	b := New(store, nil)

	ctx := &mockContext{contextType: "tmux", json: []byte(`{}`)}

	err := b.EmitWithContext("test123", "test message", ctx)
	if err != nil {
		t.Fatalf("EmitWithContext() error = %v", err)
	}

	if store.states["test123"] != "test message" {
		t.Errorf("EmitWithContext() message = %q, want %q", store.states["test123"], "test message")
	}
}

func TestBeacon_Silence_WithContextStore(t *testing.T) {
	store := newMockStore()
	store.states["test123"] = "test message"
	contextStore := newMockContextStore()
	contextStore.contexts["test123"] = &mockContext{contextType: "tmux"}
	b := NewWithContextStore(store, contextStore, nil)

	err := b.Silence("test123")
	if err != nil {
		t.Fatalf("Silence() error = %v", err)
	}

	if _, exists := store.states["test123"]; exists {
		t.Error("Silence() state still exists")
	}

	if _, exists := contextStore.contexts["test123"]; exists {
		t.Error("Silence() context still exists")
	}
}

func TestBeacon_Silence_ContextStoreError(t *testing.T) {
	store := newMockStore()
	store.states["test123"] = "test message"
	contextStore := newMockContextStore()
	contextStore.delErr = errors.New("context delete error")
	b := NewWithContextStore(store, contextStore, nil)

	err := b.Silence("test123")
	if err == nil {
		t.Error("Silence() expected error, got nil")
	}
}
