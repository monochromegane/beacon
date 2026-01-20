package beacon

import (
	"bytes"
	"errors"
	"testing"

	"github.com/monochromegane/beacon/internal/context"
)

// mockStore is a mock implementation of Store for testing.
type mockStore struct {
	states   map[int]string
	writeErr error
	delErr   error
	listErr  error
}

func newMockStore() *mockStore {
	return &mockStore{
		states: make(map[int]string),
	}
}

func (m *mockStore) Write(pid int, message string) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.states[pid] = message
	return nil
}

func (m *mockStore) Delete(pid int) error {
	if m.delErr != nil {
		return m.delErr
	}
	delete(m.states, pid)
	return nil
}

func (m *mockStore) List() ([]State, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var states []State
	for pid, msg := range m.states {
		states = append(states, State{PID: pid, Message: msg})
	}
	return states, nil
}

// mockContextStore is a mock implementation of context.ContextStore for testing.
type mockContextStore struct {
	contexts map[int]context.Context
	writeErr error
	delErr   error
}

func newMockContextStore() *mockContextStore {
	return &mockContextStore{
		contexts: make(map[int]context.Context),
	}
}

func (m *mockContextStore) Write(pid int, ctx context.Context) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.contexts[pid] = ctx
	return nil
}

func (m *mockContextStore) Delete(pid int) error {
	if m.delErr != nil {
		return m.delErr
	}
	delete(m.contexts, pid)
	return nil
}

func TestBeacon_Emit(t *testing.T) {
	store := newMockStore()
	b := New(store, nil)

	err := b.Emit(12345, "test message")
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if store.states[12345] != "test message" {
		t.Errorf("Emit() message = %q, want %q", store.states[12345], "test message")
	}
}

func TestBeacon_Emit_Error(t *testing.T) {
	store := newMockStore()
	store.writeErr = errors.New("write error")
	b := New(store, nil)

	err := b.Emit(12345, "test message")
	if err == nil {
		t.Error("Emit() expected error, got nil")
	}
}

func TestBeacon_Silence(t *testing.T) {
	store := newMockStore()
	store.states[12345] = "test message"
	b := New(store, nil)

	err := b.Silence(12345)
	if err != nil {
		t.Fatalf("Silence() error = %v", err)
	}

	if _, exists := store.states[12345]; exists {
		t.Error("Silence() state still exists")
	}
}

func TestBeacon_Silence_Error(t *testing.T) {
	store := newMockStore()
	store.delErr = errors.New("delete error")
	b := New(store, nil)

	err := b.Silence(12345)
	if err == nil {
		t.Error("Silence() expected error, got nil")
	}
}

func TestBeacon_List(t *testing.T) {
	store := newMockStore()
	store.states[12345] = "message 1"
	var buf bytes.Buffer
	b := New(store, &buf)

	err := b.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	output := buf.String()
	expected := "12345\tmessage 1\n"
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

	err := b.EmitWithContext(12345, "test message", ctx)
	if err != nil {
		t.Fatalf("EmitWithContext() error = %v", err)
	}

	if store.states[12345] != "test message" {
		t.Errorf("EmitWithContext() message = %q, want %q", store.states[12345], "test message")
	}

	if contextStore.contexts[12345] == nil {
		t.Error("EmitWithContext() context not saved")
	}
}

func TestBeacon_EmitWithContext_StoreError(t *testing.T) {
	store := newMockStore()
	store.writeErr = errors.New("write error")
	contextStore := newMockContextStore()
	b := NewWithContextStore(store, contextStore, nil)

	ctx := &mockContext{contextType: "tmux", json: []byte(`{}`)}

	err := b.EmitWithContext(12345, "test message", ctx)
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

	err := b.EmitWithContext(12345, "test message", ctx)
	if err == nil {
		t.Error("EmitWithContext() expected error, got nil")
	}
}

func TestBeacon_EmitWithContext_NilContextStore(t *testing.T) {
	store := newMockStore()
	b := New(store, nil)

	ctx := &mockContext{contextType: "tmux", json: []byte(`{}`)}

	err := b.EmitWithContext(12345, "test message", ctx)
	if err != nil {
		t.Fatalf("EmitWithContext() error = %v", err)
	}

	if store.states[12345] != "test message" {
		t.Errorf("EmitWithContext() message = %q, want %q", store.states[12345], "test message")
	}
}

func TestBeacon_Silence_WithContextStore(t *testing.T) {
	store := newMockStore()
	store.states[12345] = "test message"
	contextStore := newMockContextStore()
	contextStore.contexts[12345] = &mockContext{contextType: "tmux"}
	b := NewWithContextStore(store, contextStore, nil)

	err := b.Silence(12345)
	if err != nil {
		t.Fatalf("Silence() error = %v", err)
	}

	if _, exists := store.states[12345]; exists {
		t.Error("Silence() state still exists")
	}

	if _, exists := contextStore.contexts[12345]; exists {
		t.Error("Silence() context still exists")
	}
}

func TestBeacon_Silence_ContextStoreError(t *testing.T) {
	store := newMockStore()
	store.states[12345] = "test message"
	contextStore := newMockContextStore()
	contextStore.delErr = errors.New("context delete error")
	b := NewWithContextStore(store, contextStore, nil)

	err := b.Silence(12345)
	if err == nil {
		t.Error("Silence() expected error, got nil")
	}
}
