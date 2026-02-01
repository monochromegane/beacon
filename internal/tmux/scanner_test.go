package tmux

import (
	"testing"

	"github.com/monochromegane/beacon/internal/signal"
)

type mockScannerExecutor struct {
	outputs map[string][]byte
	err     error
}

func (m *mockScannerExecutor) Execute(name string, args ...string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Build a key from the command
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	if output, ok := m.outputs[key]; ok {
		return output, nil
	}
	return []byte{}, nil
}

func TestScanner_ScanWindows(t *testing.T) {
	executor := &mockScannerExecutor{
		outputs: map[string][]byte{
			"tmux list-windows -F #{window_index}\t#{window_name}\t#{window_id}": []byte("0\tbash\t@0\n1\tvim\t@1\n"),
			"tmux display-message -p #{session_name}":                            []byte("main\n"),
		},
	}
	scanner := NewScannerWithExecutor(executor)

	signals := []*signal.Signal{
		{
			SessionID:  "test1",
			SignalType: "claude",
			State:      signal.StateRunning,
			Message:    "claude:running",
			Environment: &signal.Environment{
				Type:        "tmux",
				SessionName: "main",
				WindowIndex: 0,
				PaneIndex:   0,
				PaneID:      "%0",
			},
		},
	}

	windows, err := scanner.ScanWindows(signals)
	if err != nil {
		t.Fatalf("ScanWindows() error = %v", err)
	}

	if len(windows) != 2 {
		t.Fatalf("ScanWindows() returned %d windows, want 2", len(windows))
	}

	// First window should have the signal
	if len(windows[0].Signals) != 1 {
		t.Errorf("windows[0].Signals = %d, want 1", len(windows[0].Signals))
	}
	if windows[0].Signals[0].SessionID != "test1" {
		t.Errorf("windows[0].Signals[0].SessionID = %q, want %q", windows[0].Signals[0].SessionID, "test1")
	}

	// Second window should have no signals
	if len(windows[1].Signals) != 0 {
		t.Errorf("windows[1].Signals = %d, want 0", len(windows[1].Signals))
	}
}

func TestScanner_ScanSessions(t *testing.T) {
	executor := &mockScannerExecutor{
		outputs: map[string][]byte{
			"tmux list-sessions -F #{session_name}":                                      []byte("main\nwork\n"),
			"tmux list-windows -t main -F #{window_index}\t#{window_name}\t#{window_id}": []byte("0\tbash\t@0\n"),
			"tmux list-windows -t work -F #{window_index}\t#{window_name}\t#{window_id}": []byte("0\tcode\t@1\n"),
		},
	}
	scanner := NewScannerWithExecutor(executor)

	signals := []*signal.Signal{
		{
			SessionID:  "test1",
			SignalType: "claude",
			State:      signal.StateRunning,
			Message:    "claude:running",
			Environment: &signal.Environment{
				Type:        "tmux",
				SessionName: "work",
				WindowIndex: 0,
				PaneIndex:   0,
				PaneID:      "%0",
			},
		},
	}

	windows, err := scanner.ScanSessions(signals)
	if err != nil {
		t.Fatalf("ScanSessions() error = %v", err)
	}

	if len(windows) != 2 {
		t.Fatalf("ScanSessions() returned %d windows, want 2", len(windows))
	}

	// Find the work session window
	var workWindow *WindowInfo
	for i := range windows {
		if windows[i].SessionName == "work" {
			workWindow = &windows[i]
			break
		}
	}
	if workWindow == nil {
		t.Fatal("work session window not found")
	}
	if len(workWindow.Signals) != 1 {
		t.Errorf("work window signals = %d, want 1", len(workWindow.Signals))
	}
}

func TestScanner_ScanWindows_NoSignals(t *testing.T) {
	executor := &mockScannerExecutor{
		outputs: map[string][]byte{
			"tmux list-windows -F #{window_index}\t#{window_name}\t#{window_id}": []byte("0\tbash\t@0\n"),
			"tmux display-message -p #{session_name}":                            []byte("main\n"),
		},
	}
	scanner := NewScannerWithExecutor(executor)

	windows, err := scanner.ScanWindows(nil)
	if err != nil {
		t.Fatalf("ScanWindows() error = %v", err)
	}

	if len(windows) != 1 {
		t.Fatalf("ScanWindows() returned %d windows, want 1", len(windows))
	}
	if len(windows[0].Signals) != 0 {
		t.Errorf("windows[0].Signals = %d, want 0", len(windows[0].Signals))
	}
}
