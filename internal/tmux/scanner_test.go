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
			"tmux list-windows -F #{window_index}\t#{window_name}\t#{window_id}\t#{window_panes}": []byte("0\tbash\t@0\t2\n1\tvim\t@1\t1\n"),
			"tmux display-message -p #{session_name}":                                             []byte("main\n"),
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

	// First window should have the signal and pane count
	if len(windows[0].Signals) != 1 {
		t.Errorf("windows[0].Signals = %d, want 1", len(windows[0].Signals))
	}
	if windows[0].Signals[0].SessionID != "test1" {
		t.Errorf("windows[0].Signals[0].SessionID = %q, want %q", windows[0].Signals[0].SessionID, "test1")
	}
	if windows[0].PaneCount != 2 {
		t.Errorf("windows[0].PaneCount = %d, want 2", windows[0].PaneCount)
	}

	// Second window should have no signals and pane count of 1
	if len(windows[1].Signals) != 0 {
		t.Errorf("windows[1].Signals = %d, want 0", len(windows[1].Signals))
	}
	if windows[1].PaneCount != 1 {
		t.Errorf("windows[1].PaneCount = %d, want 1", windows[1].PaneCount)
	}
}

func TestScanner_ScanSessions(t *testing.T) {
	executor := &mockScannerExecutor{
		outputs: map[string][]byte{
			"tmux list-sessions -F #{session_name}":                                                       []byte("main\nwork\n"),
			"tmux list-windows -t main -F #{window_index}\t#{window_name}\t#{window_id}\t#{window_panes}": []byte("0\tbash\t@0\t1\n"),
			"tmux list-windows -t work -F #{window_index}\t#{window_name}\t#{window_id}\t#{window_panes}": []byte("0\tcode\t@1\t3\n"),
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
	if workWindow.PaneCount != 3 {
		t.Errorf("work window pane count = %d, want 3", workWindow.PaneCount)
	}
}

func TestScanner_ScanWindows_NoSignals(t *testing.T) {
	executor := &mockScannerExecutor{
		outputs: map[string][]byte{
			"tmux list-windows -F #{window_index}\t#{window_name}\t#{window_id}\t#{window_panes}": []byte("0\tbash\t@0\t1\n"),
			"tmux display-message -p #{session_name}":                                             []byte("main\n"),
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

func TestScanner_ScanSessionsAggregated(t *testing.T) {
	executor := &mockScannerExecutor{
		outputs: map[string][]byte{
			"tmux list-sessions -F #{session_name}\t#{session_windows}": []byte("main\t2\nwork\t3\n"),
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
		{
			SessionID:  "test2",
			SignalType: "claude",
			State:      signal.StateIdle,
			Message:    "claude:idle",
			Environment: &signal.Environment{
				Type:        "tmux",
				SessionName: "work",
				WindowIndex: 1,
				PaneIndex:   0,
				PaneID:      "%1",
			},
		},
	}

	sessions, err := scanner.ScanSessionsAggregated(signals)
	if err != nil {
		t.Fatalf("ScanSessionsAggregated() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("ScanSessionsAggregated() returned %d sessions, want 2", len(sessions))
	}

	// Find main session
	var mainSession, workSession *SessionInfo
	for i := range sessions {
		if sessions[i].SessionName == "main" {
			mainSession = &sessions[i]
		}
		if sessions[i].SessionName == "work" {
			workSession = &sessions[i]
		}
	}

	if mainSession == nil {
		t.Fatal("main session not found")
	}
	if mainSession.WindowCount != 2 {
		t.Errorf("main session window count = %d, want 2", mainSession.WindowCount)
	}
	if len(mainSession.Signals) != 0 {
		t.Errorf("main session signals = %d, want 0", len(mainSession.Signals))
	}

	if workSession == nil {
		t.Fatal("work session not found")
	}
	if workSession.WindowCount != 3 {
		t.Errorf("work session window count = %d, want 3", workSession.WindowCount)
	}
	if len(workSession.Signals) != 2 {
		t.Errorf("work session signals = %d, want 2", len(workSession.Signals))
	}
}
