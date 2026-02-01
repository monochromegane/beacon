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
			PaneTitle:   "Building",
		},
	}

	executor := &mockTmuxExecutor{
		outputs: map[string][]byte{
			"tmux list-windows -F #{window_index}\t#{window_name}\t#{window_id}\t#{window_panes}": []byte("0\tbash\t@0\t2\n1\tvim\t@1\t1\n"),
			"tmux display-message -p #{session_name}":                                             []byte("main\n"),
		},
	}
	scanner := tmux.NewScannerWithExecutor(executor)

	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.tmuxScanner = scanner
	cli.out = &buf
	cli.in = strings.NewReader("")

	err := cli.Execute([]string{"scan", "--color=never"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	// New format: {window_index}: {window_name} ({pane_count} panes) | {state}: "{title}"
	if !strings.Contains(output, `0: bash (2 panes) | running: "Building"`) {
		t.Errorf("Output = %q, want to contain %q", output, `0: bash (2 panes) | running: "Building"`)
	}
	if !strings.Contains(output, "1: vim (1 panes)") {
		t.Errorf("Output = %q, want to contain %q", output, "1: vim (1 panes)")
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
			"tmux list-windows -F #{window_index}\t#{window_name}\t#{window_id}\t#{window_panes}": []byte("0\tbash\t@0\t2\n"),
			"tmux display-message -p #{session_name}":                                             []byte("main\n"),
		},
	}
	scanner := tmux.NewScannerWithExecutor(executor)

	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.tmuxScanner = scanner
	cli.out = &buf
	cli.in = strings.NewReader("")

	err := cli.Execute([]string{"scan", "--template", "{{.WindowName}} ({{.PaneCount}} panes):{{range .Signals}}{{.State}}{{end}}"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "bash (2 panes):running") {
		t.Errorf("Output = %q, want to contain %q", output, "bash (2 panes):running")
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
			PaneTitle:   "Building",
		},
	}

	executor := &mockTmuxExecutor{
		outputs: map[string][]byte{
			"tmux list-sessions -F #{session_name}\t#{session_windows}": []byte("main\t1\nwork\t2\n"),
		},
	}
	scanner := tmux.NewScannerWithExecutor(executor)

	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.tmuxScanner = scanner
	cli.out = &buf
	cli.in = strings.NewReader("")

	err := cli.Execute([]string{"scan", "--scope", "session", "--color=never"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	// work session should have signal with new format
	if !strings.Contains(output, `work: 2 windows | running: "Building"`) {
		t.Errorf("Output = %q, want to contain %q", output, `work: 2 windows | running: "Building"`)
	}
	// main session should have no signals
	if !strings.Contains(output, "main: 1 windows") {
		t.Errorf("Output = %q, want to contain %q", output, "main: 1 windows")
	}
}

func TestCLI_Scan_SessionScope_WithTemplate(t *testing.T) {
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
			"tmux list-sessions -F #{session_name}\t#{session_windows}": []byte("work\t3\n"),
		},
	}
	scanner := tmux.NewScannerWithExecutor(executor)

	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.tmuxScanner = scanner
	cli.out = &buf
	cli.in = strings.NewReader("")

	err := cli.Execute([]string{"scan", "--scope", "session", "--template", "{{.SessionName}}: {{.WindowCount}} windows"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "work: 3 windows") {
		t.Errorf("Output = %q, want to contain %q", output, "work: 3 windows")
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

type mockContextProvider struct {
	env *signal.Environment
}

func (m *mockContextProvider) GetEnvironment() (*signal.Environment, error) {
	return m.env, nil
}

func newTestCLIWithTmux(store *mockSignalStore, input string, provider *mockContextProvider) (*CLI, *bytes.Buffer) {
	var buf bytes.Buffer
	cli := NewCLI()
	cli.signalStore = store
	cli.tmuxContextProvider = provider
	cli.out = &buf
	cli.in = strings.NewReader(input)
	return cli, &buf
}

func TestCLI_Emit_PreservesEnvironment_OnStop(t *testing.T) {
	store := newMockSignalStore()
	// Simulate existing signal from SessionStart (window 0)
	store.signals["claude_test123"] = &signal.Signal{
		SessionID:  "test123",
		SignalType: "claude",
		State:      signal.StateStarted,
		Environment: &signal.Environment{
			Type:        "tmux",
			SessionName: "main",
			WindowIndex: 0,
			PaneIndex:   0,
			PaneID:      "%0",
			PaneTitle:   "Initial",
		},
	}

	// Mock provider returns window 1 (simulating user moved to different window)
	provider := &mockContextProvider{
		env: &signal.Environment{
			Type:        "tmux",
			SessionName: "main",
			WindowIndex: 1, // Different window!
			PaneIndex:   0,
			PaneID:      "%1",
			PaneTitle:   "CurrentTitle",
		},
	}

	cli, _ := newTestCLIWithTmux(store, `{"session_id":"test123","hook_event_name":"Stop"}`, provider)

	err := cli.Execute([]string{"emit"})
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
	if sig.Environment == nil {
		t.Fatal("Environment is nil")
	}
	// Window index should be preserved from original signal (0), not current (1)
	if sig.Environment.WindowIndex != 0 {
		t.Errorf("WindowIndex = %d, want %d (should be preserved)", sig.Environment.WindowIndex, 0)
	}
	// PaneTitle should be updated to current value
	if sig.Environment.PaneTitle != "CurrentTitle" {
		t.Errorf("PaneTitle = %q, want %q (should be updated)", sig.Environment.PaneTitle, "CurrentTitle")
	}
}

func TestCLI_Emit_UpdatesPaneTitle_OnStop(t *testing.T) {
	store := newMockSignalStore()
	store.signals["claude_test123"] = &signal.Signal{
		SessionID:  "test123",
		SignalType: "claude",
		State:      signal.StateRunning,
		Environment: &signal.Environment{
			Type:        "tmux",
			SessionName: "main",
			WindowIndex: 0,
			PaneIndex:   0,
			PaneID:      "%0",
			PaneTitle:   "OldTitle",
		},
	}

	provider := &mockContextProvider{
		env: &signal.Environment{
			Type:        "tmux",
			SessionName: "main",
			WindowIndex: 0,
			PaneIndex:   0,
			PaneID:      "%0",
			PaneTitle:   "NewTitle",
		},
	}

	cli, _ := newTestCLIWithTmux(store, `{"session_id":"test123","hook_event_name":"Stop"}`, provider)

	err := cli.Execute([]string{"emit"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	sig := store.signals["claude_test123"]
	if sig == nil {
		t.Fatal("Signal not stored")
	}
	if sig.Environment.PaneTitle != "NewTitle" {
		t.Errorf("PaneTitle = %q, want %q", sig.Environment.PaneTitle, "NewTitle")
	}
}

func TestCLI_Emit_FreshEnvironment_OnSessionStart(t *testing.T) {
	store := newMockSignalStore()
	// No existing signal

	provider := &mockContextProvider{
		env: &signal.Environment{
			Type:        "tmux",
			SessionName: "main",
			WindowIndex: 2,
			PaneIndex:   1,
			PaneID:      "%5",
			PaneTitle:   "Fresh",
		},
	}

	cli, _ := newTestCLIWithTmux(store, `{"session_id":"test123","hook_event_name":"SessionStart","source":"cli"}`, provider)

	err := cli.Execute([]string{"emit"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	sig := store.signals["claude_test123"]
	if sig == nil {
		t.Fatal("Signal not stored")
	}
	if sig.Environment == nil {
		t.Fatal("Environment is nil")
	}
	// Should use fresh environment values
	if sig.Environment.WindowIndex != 2 {
		t.Errorf("WindowIndex = %d, want %d", sig.Environment.WindowIndex, 2)
	}
	if sig.Environment.PaneIndex != 1 {
		t.Errorf("PaneIndex = %d, want %d", sig.Environment.PaneIndex, 1)
	}
	if sig.Environment.PaneID != "%5" {
		t.Errorf("PaneID = %q, want %q", sig.Environment.PaneID, "%5")
	}
	if sig.Environment.PaneTitle != "Fresh" {
		t.Errorf("PaneTitle = %q, want %q", sig.Environment.PaneTitle, "Fresh")
	}
}
