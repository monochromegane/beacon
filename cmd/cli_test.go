package cmd

import (
	"bytes"
	"testing"

	"github.com/monochromegane/beacon/internal/beacon"
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

func TestCLI_Emit(t *testing.T) {
	store := newMockStore()
	cli := NewCLI(12345)
	cli.store = store

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
	cli := NewCLI(12345)
	cli.store = store

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
	var buf bytes.Buffer
	cli := NewCLI(99999)
	cli.store = store
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
