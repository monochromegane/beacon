package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/monochromegane/beacon/internal/beacon"
	"github.com/monochromegane/beacon/internal/context"
)

func TestNewCLI(t *testing.T) {
	cli := NewCLI(12345)
	if cli == nil {
		t.Error("NewCLI() returned nil")
	}
	if cli.ppid != 12345 {
		t.Errorf("NewCLI() ppid = %d, want 12345", cli.ppid)
	}
}

type mockStore struct {
	states map[int]string
}

func newMockStore() *mockStore {
	return &mockStore{states: make(map[int]string)}
}

func (m *mockStore) Write(pid int, message string) error {
	m.states[pid] = message
	return nil
}

func (m *mockStore) Delete(pid int) error {
	delete(m.states, pid)
	return nil
}

func (m *mockStore) List() ([]beacon.State, error) {
	var states []beacon.State
	for pid, msg := range m.states {
		states = append(states, beacon.State{PID: pid, Message: msg})
	}
	return states, nil
}

type mockContextStore struct {
	contexts map[int]context.Context
}

func newMockContextStore() *mockContextStore {
	return &mockContextStore{contexts: make(map[int]context.Context)}
}

func (m *mockContextStore) Write(pid int, ctx context.Context) error {
	m.contexts[pid] = ctx
	return nil
}

func (m *mockContextStore) Delete(pid int) error {
	delete(m.contexts, pid)
	return nil
}

func (m *mockContextStore) Read(pid int) ([]byte, error) {
	ctx, ok := m.contexts[pid]
	if !ok {
		return nil, os.ErrNotExist
	}
	return ctx.ToJSON()
}

func TestCLI_Emit(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	cli := NewCLI(12345)
	cli.store = store
	cli.contextStore = contextStore

	err := cli.Execute([]string{"emit", "test message"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if store.states[12345] != "test message" {
		t.Errorf("Emit message = %q, want %q", store.states[12345], "test message")
	}
}

func TestCLI_Silence(t *testing.T) {
	store := newMockStore()
	store.states[12345] = "existing message"
	contextStore := newMockContextStore()
	cli := NewCLI(12345)
	cli.store = store
	cli.contextStore = contextStore

	err := cli.Execute([]string{"silence"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, exists := store.states[12345]; exists {
		t.Error("Silence did not delete the state")
	}
}

func TestCLI_List(t *testing.T) {
	store := newMockStore()
	store.states[12345] = "message 1"
	contextStore := newMockContextStore()
	var buf bytes.Buffer
	cli := NewCLI(99999)
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"list"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expected := "12345\tmessage 1\n"
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
	cli := NewCLI(12345)
	cli.store = store
	cli.contextStore = contextStore

	err := cli.Execute([]string{"emit", "--context", "tmux", "test message"})
	if err == nil {
		t.Error("Execute() expected error when not in tmux, got nil")
	}
}

func TestCLI_Context_JSON(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	contextStore.contexts[12345] = &context.TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
	}
	var buf bytes.Buffer
	cli := NewCLI(99999)
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"context", "12345"})
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
	contextStore.contexts[12345] = &context.TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
	}
	var buf bytes.Buffer
	cli := NewCLI(99999)
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"context", "--template", "{{.session_name}}:{{.pane_id}}", "12345"})
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
	cli := NewCLI(99999)
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"context", "99999"})
	if err == nil {
		t.Error("Execute() expected error for non-existent context, got nil")
	}
}

func TestCLI_Context_InvalidTemplate(t *testing.T) {
	store := newMockStore()
	contextStore := newMockContextStore()
	contextStore.contexts[12345] = &context.TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
	}
	var buf bytes.Buffer
	cli := NewCLI(99999)
	cli.store = store
	cli.contextStore = contextStore
	cli.out = &buf

	err := cli.Execute([]string{"context", "--template", "{{.invalid", "12345"})
	if err == nil {
		t.Error("Execute() expected error for invalid template, got nil")
	}
}
