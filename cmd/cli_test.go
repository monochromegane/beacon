package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/monochromegane/beacon/internal/beacon"
	"github.com/monochromegane/beacon/internal/context"
)

func TestNewCLI(t *testing.T) {
	cli := NewCLI()
	if cli == nil {
		t.Error("NewCLI() returned nil")
	}
}

type mockStore struct {
	states map[string]string
}

func newMockStore() *mockStore {
	return &mockStore{states: make(map[string]string)}
}

func (m *mockStore) Write(id string, message string) error {
	m.states[id] = message
	return nil
}

func (m *mockStore) Delete(id string) error {
	delete(m.states, id)
	return nil
}

func (m *mockStore) List() ([]beacon.State, error) {
	var states []beacon.State
	for id, msg := range m.states {
		states = append(states, beacon.State{ID: id, Message: msg})
	}
	return states, nil
}

type mockContextStore struct {
	contexts map[string]context.Context
}

func newMockContextStore() *mockContextStore {
	return &mockContextStore{contexts: make(map[string]context.Context)}
}

func (m *mockContextStore) Write(id string, ctx context.Context) error {
	m.contexts[id] = ctx
	return nil
}

func (m *mockContextStore) Delete(id string) error {
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

func TestCLI_Emit(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	cli := NewCLI()
	cli.store = store
	cli.contextStore = contextStore

	err := cli.Execute([]string{"emit", "--id", "test123", "test message"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if store.states["test123"] != "test message" {
		t.Errorf("Emit message = %q, want %q", store.states["test123"], "test message")
	}
}

func TestCLI_Silence(t *testing.T) {
	store := newMockStore()
	store.states["test123"] = "existing message"
	contextStore := newMockContextStore()
	cli := NewCLI()
	cli.store = store
	cli.contextStore = contextStore

	err := cli.Execute([]string{"silence", "--id", "test123"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, exists := store.states["test123"]; exists {
		t.Error("Silence did not delete the state")
	}
}

func TestCLI_List(t *testing.T) {
	store := newMockStore()
	store.states["test123"] = "message 1"
	contextStore := newMockContextStore()
	var buf bytes.Buffer
	cli := NewCLI()
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"list"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expected := "test123\tmessage 1\n"
	if buf.String() != expected {
		t.Errorf("List output = %q, want %q", buf.String(), expected)
	}
}

func TestCLI_Emit_WithContext_NotInTmux(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	os.Unsetenv("TMUX")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		}
	}()

	store := newMockStore()
	contextStore := newMockContextStore()
	cli := NewCLI()
	cli.store = store
	cli.contextStore = contextStore

	err := cli.Execute([]string{"emit", "--id", "test123", "--context", "tmux", "test message"})
	if err == nil {
		t.Error("Execute() expected error when not in tmux, got nil")
	}
}

func TestCLI_Context_JSON(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	contextStore.contexts["test123"] = &context.TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
	}
	var buf bytes.Buffer
	cli := NewCLI()
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"context", "test123"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expected := `{"session_name":"main","window_index":0,"pane_index":1,"pane_id":"%2"}` + "\n"
	if buf.String() != expected {
		t.Errorf("Context output = %q, want %q", buf.String(), expected)
	}
}

func TestCLI_Context_Template(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	contextStore.contexts["test123"] = &context.TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
	}
	var buf bytes.Buffer
	cli := NewCLI()
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"context", "--template", "{{.session_name}}:{{.pane_id}}", "test123"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expected := "main:%2\n"
	if buf.String() != expected {
		t.Errorf("Context output = %q, want %q", buf.String(), expected)
	}
}

func TestCLI_Context_NotFound(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	var buf bytes.Buffer
	cli := NewCLI()
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"context", "nonexistent"})
	if err == nil {
		t.Error("Execute() expected error for non-existent context, got nil")
	}
}

func TestCLI_Context_InvalidTemplate(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	contextStore.contexts["test123"] = &context.TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
	}
	var buf bytes.Buffer
	cli := NewCLI()
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"context", "--template", "{{.invalid", "test123"})
	if err == nil {
		t.Error("Execute() expected error for invalid template, got nil")
	}
}
