package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/monochromegane/beacon/internal/signal"
	"github.com/monochromegane/beacon/internal/tmux"
)

func TestNewCLI(t *testing.T) {
	cli := NewCLI()
	if cli == nil {
		t.Error("NewCLI() returned nil")
	}
}

type mockSignalStore struct {
	signals map[string]*signal.Signal
}

func newMockSignalStore() *mockSignalStore {
	return &mockSignalStore{signals: make(map[string]*signal.Signal)}
}

func (m *mockSignalStore) Write(sig *signal.Signal) error {
	key := sig.SignalType + "_" + sig.SessionID
	m.signals[key] = sig
	return nil
}

func (m *mockSignalStore) Delete(signalType, sessionID string) error {
	key := signalType + "_" + sessionID
	delete(m.signals, key)
	return nil
}

func (m *mockSignalStore) Read(signalType, sessionID string) (*signal.Signal, error) {
	key := signalType + "_" + sessionID
	return m.signals[key], nil
}

func (m *mockSignalStore) List(signalType string) ([]*signal.Signal, error) {
	var result []*signal.Signal
	for key, sig := range m.signals {
		if strings.HasPrefix(key, signalType+"_") {
			result = append(result, sig)
		}
	}
	return result, nil
}

type mockTmuxExecutor struct {
	outputs map[string][]byte
}

func (m *mockTmuxExecutor) Execute(name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	if output, ok := m.outputs[key]; ok {
		return output, nil
	}
	return []byte{}, nil
}

func newTestCLI(store *mockSignalStore, input string) (*CLI, *bytes.Buffer) {
	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.out = &buf
	cli.in = strings.NewReader(input)
	return cli, &buf
}

func TestCLI_Emit_SessionStart(t *testing.T) {
	store := newMockSignalStore()
	cli, _ := newTestCLI(store, `{"session_id":"test123","hook_event_name":"SessionStart","source":"cli"}`)

	err := cli.Execute([]string{"emit", "--env", ""})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	sig := store.signals["claude_test123"]
	if sig == nil {
		t.Fatal("Signal not stored")
	}
	if sig.State != signal.StateStarted {
		t.Errorf("State = %q, want %q", sig.State, signal.StateStarted)
	}
	if sig.Message != "claude:started:cli" {
		t.Errorf("Message = %q, want %q", sig.Message, "claude:started:cli")
	}
}

func TestCLI_Emit_UserPromptSubmit(t *testing.T) {
	store := newMockSignalStore()
	cli, _ := newTestCLI(store, `{"session_id":"test123","hook_event_name":"UserPromptSubmit"}`)

	err := cli.Execute([]string{"emit", "--env", ""})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	sig := store.signals["claude_test123"]
	if sig == nil {
		t.Fatal("Signal not stored")
	}
	if sig.State != signal.StateRunning {
		t.Errorf("State = %q, want %q", sig.State, signal.StateRunning)
	}
}

func TestCLI_Emit_Stop(t *testing.T) {
	store := newMockSignalStore()
	cli, _ := newTestCLI(store, `{"session_id":"test123","hook_event_name":"Stop"}`)

	err := cli.Execute([]string{"emit", "--env", ""})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	sig := store.signals["claude_test123"]
	if sig == nil {
		t.Fatal("Signal not stored")
	}
	if sig.State != signal.StateIdle {
		t.Errorf("State = %q, want %q", sig.State, signal.StateIdle)
	}
}

func TestCLI_Emit_SessionEnd(t *testing.T) {
	store := newMockSignalStore()
	store.signals["claude_test123"] = &signal.Signal{
		SessionID:  "test123",
		SignalType: "claude",
		State:      signal.StateRunning,
	}

	cli, _ := newTestCLI(store, `{"session_id":"test123","hook_event_name":"SessionEnd"}`)

	err := cli.Execute([]string{"emit", "--env", ""})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, exists := store.signals["claude_test123"]; exists {
		t.Error("Signal should have been deleted")
	}
}

func TestCLI_Emit_WithCustomMessage(t *testing.T) {
	store := newMockSignalStore()
	cli, _ := newTestCLI(store, `{"session_id":"test123","hook_event_name":"UserPromptSubmit"}`)

	err := cli.Execute([]string{"emit", "--env", "", "custom message"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	sig := store.signals["claude_test123"]
	if sig == nil {
		t.Fatal("Signal not stored")
	}
	if sig.CustomMessage != "custom message" {
		t.Errorf("CustomMessage = %q, want %q", sig.CustomMessage, "custom message")
	}
}

func TestCLI_Emit_Notification_PermissionPrompt(t *testing.T) {
	store := newMockSignalStore()
	cli, _ := newTestCLI(store, `{"session_id":"test123","hook_event_name":"Notification","notification_type":"permission_prompt"}`)

	err := cli.Execute([]string{"emit", "--env", ""})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	sig := store.signals["claude_test123"]
	if sig == nil {
		t.Fatal("Signal not stored")
	}
	if sig.State != signal.StateWaiting {
		t.Errorf("State = %q, want %q", sig.State, signal.StateWaiting)
	}
}

func TestCLI_Scan_Default(t *testing.T) {
	store := newMockSignalStore()
	store.signals["claude_test1"] = &signal.Signal{
		SessionID:  "test1",
		SignalType: "claude",
		State:      signal.StateRunning,
		Message:    "claude:running",
		UpdatedAt:  time.Now(),
		Environment: &signal.Environment{
			Type:        "tmux",
			SessionName: "main",
			WindowIndex: 0,
			PaneIndex:   0,
			PaneID:      "%0",
		},
	}

	executor := &mockTmuxExecutor{
		outputs: map[string][]byte{
			"tmux list-windows -F #{window_index}\t#{window_name}\t#{window_id}": []byte("0\tbash\t@0\n1\tvim\t@1\n"),
			"tmux display-message -p #{session_name}":                            []byte("main\n"),
		},
	}
	scanner := tmux.NewScannerWithExecutor(executor)

	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.tmuxScanner = scanner
	cli.out = &buf
	cli.in = strings.NewReader("")

	err := cli.Execute([]string{"scan"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "0: bash:running") {
		t.Errorf("Output = %q, want to contain %q", output, "0: bash:running")
	}
	if !strings.Contains(output, "1: vim:") {
		t.Errorf("Output = %q, want to contain %q", output, "1: vim:")
	}
}

func TestCLI_Scan_WithTemplate(t *testing.T) {
	store := newMockSignalStore()
	store.signals["claude_test1"] = &signal.Signal{
		SessionID:  "test1",
		SignalType: "claude",
		State:      signal.StateRunning,
		Message:    "claude:running",
		UpdatedAt:  time.Now(),
		Environment: &signal.Environment{
			Type:        "tmux",
			SessionName: "main",
			WindowIndex: 0,
			PaneIndex:   0,
			PaneID:      "%0",
		},
	}

	executor := &mockTmuxExecutor{
		outputs: map[string][]byte{
			"tmux list-windows -F #{window_index}\t#{window_name}\t#{window_id}": []byte("0\tbash\t@0\n"),
			"tmux display-message -p #{session_name}":                            []byte("main\n"),
		},
	}
	scanner := tmux.NewScannerWithExecutor(executor)

	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.tmuxScanner = scanner
	cli.out = &buf
	cli.in = strings.NewReader("")

	err := cli.Execute([]string{"scan", "--template", "{{.WindowName}}:{{range .Signals}}{{.State}}{{end}}"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "bash:running") {
		t.Errorf("Output = %q, want to contain %q", output, "bash:running")
	}
}

func TestCLI_Scan_SessionScope(t *testing.T) {
	store := newMockSignalStore()
	store.signals["claude_test1"] = &signal.Signal{
		SessionID:  "test1",
		SignalType: "claude",
		State:      signal.StateRunning,
		Message:    "claude:running",
		UpdatedAt:  time.Now(),
		Environment: &signal.Environment{
			Type:        "tmux",
			SessionName: "work",
			WindowIndex: 0,
			PaneIndex:   0,
			PaneID:      "%0",
		},
	}

	executor := &mockTmuxExecutor{
		outputs: map[string][]byte{
			"tmux list-sessions -F #{session_name}":                                      []byte("main\nwork\n"),
			"tmux list-windows -t main -F #{window_index}\t#{window_name}\t#{window_id}": []byte("0\tbash\t@0\n"),
			"tmux list-windows -t work -F #{window_index}\t#{window_name}\t#{window_id}": []byte("0\tcode\t@1\n"),
		},
	}
	scanner := tmux.NewScannerWithExecutor(executor)

	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.tmuxScanner = scanner
	cli.out = &buf
	cli.in = strings.NewReader("")

	err := cli.Execute([]string{"scan", "--scope", "session"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	// work session should have signal
	if !strings.Contains(output, "0: code:running") {
		t.Errorf("Output = %q, want to contain %q", output, "0: code:running")
	}
}

func TestCLI_Emit_InvalidJSON(t *testing.T) {
	store := newMockSignalStore()
	cli, _ := newTestCLI(store, `{invalid}`)

	err := cli.Execute([]string{"emit", "--env", ""})
	if err == nil {
		t.Error("Execute() expected error for invalid JSON, got nil")
	}
}
