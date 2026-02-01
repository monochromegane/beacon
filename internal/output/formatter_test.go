package output

import (
	"testing"

	"github.com/monochromegane/beacon/internal/tmux"
)

func TestSortByPriority(t *testing.T) {
	signals := []tmux.SignalInfo{
		{State: "running", PaneTitle: "Task 1"},
		{State: "waiting", PaneTitle: "Task 2"},
		{State: "idle", PaneTitle: "Task 3"},
		{State: "started", PaneTitle: "Task 4"},
		{State: "running", PaneTitle: "Task 5"},
	}

	sorted := SortByPriority(signals)

	// Expected order: waiting (1), idle (2), started (2), running (3), running (3)
	expected := []string{"waiting", "idle", "started", "running", "running"}
	for i, exp := range expected {
		if sorted[i].State != exp {
			t.Errorf("sorted[%d].State = %q, want %q", i, sorted[i].State, exp)
		}
	}

	// Verify original slice is not modified
	if signals[0].State != "running" {
		t.Error("Original slice was modified")
	}
}

func TestSortByPriority_PreservesOrderWithinSamePriority(t *testing.T) {
	signals := []tmux.SignalInfo{
		{State: "idle", PaneTitle: "First idle"},
		{State: "started", PaneTitle: "First started"},
		{State: "idle", PaneTitle: "Second idle"},
	}

	sorted := SortByPriority(signals)

	// idle and started have same priority (2), so relative order should be preserved
	if sorted[0].PaneTitle != "First idle" {
		t.Errorf("sorted[0].PaneTitle = %q, want %q", sorted[0].PaneTitle, "First idle")
	}
	if sorted[1].PaneTitle != "First started" {
		t.Errorf("sorted[1].PaneTitle = %q, want %q", sorted[1].PaneTitle, "First started")
	}
	if sorted[2].PaneTitle != "Second idle" {
		t.Errorf("sorted[2].PaneTitle = %q, want %q", sorted[2].PaneTitle, "Second idle")
	}
}

func TestFormatter_ColorizeState_WithColor(t *testing.T) {
	scheme := NewColorScheme(true)
	formatter := NewFormatter(scheme)

	tests := []struct {
		state    string
		expected string
	}{
		{"waiting", "\033[33mwaiting\033[0m"},
		{"idle", "\033[36midle\033[0m"},
		{"started", "\033[34mstarted\033[0m"},
		{"running", "\033[32mrunning\033[0m"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := formatter.ColorizeState(tt.state)
		if result != tt.expected {
			t.Errorf("ColorizeState(%q) = %q, want %q", tt.state, result, tt.expected)
		}
	}
}

func TestFormatter_ColorizeState_WithoutColor(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	states := []string{"waiting", "idle", "started", "running", "unknown"}
	for _, state := range states {
		result := formatter.ColorizeState(state)
		if result != state {
			t.Errorf("ColorizeState(%q) = %q, want %q (no color)", state, result, state)
		}
	}
}

func TestFormatter_FormatSignals(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	signals := []tmux.SignalInfo{
		{State: "running", PaneTitle: "Building project"},
		{State: "waiting", PaneTitle: "Reviewing PR"},
	}

	result := formatter.FormatSignals(signals)
	expected := `waiting: "Reviewing PR", running: "Building project"`
	if result != expected {
		t.Errorf("FormatSignals() = %q, want %q", result, expected)
	}
}

func TestFormatter_FormatSignals_Empty(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	result := formatter.FormatSignals(nil)
	if result != "" {
		t.Errorf("FormatSignals(nil) = %q, want empty string", result)
	}
}

func TestFormatter_FormatSignals_NoTitle(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	signals := []tmux.SignalInfo{
		{State: "running", PaneTitle: ""},
	}

	result := formatter.FormatSignals(signals)
	expected := "running"
	if result != expected {
		t.Errorf("FormatSignals() = %q, want %q", result, expected)
	}
}

func TestFormatter_FormatSignals_UsesCustomMessageWhenNoTitle(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	signals := []tmux.SignalInfo{
		{State: "running", PaneTitle: "", CustomMessage: "Custom task"},
	}

	result := formatter.FormatSignals(signals)
	expected := `running: "Custom task"`
	if result != expected {
		t.Errorf("FormatSignals() = %q, want %q", result, expected)
	}
}

func TestFormatter_FormatWindow(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	window := tmux.WindowInfo{
		WindowIndex: 0,
		WindowName:  "dev",
		PaneCount:   3,
		Signals: []tmux.SignalInfo{
			{State: "running", PaneTitle: "Building project"},
			{State: "waiting", PaneTitle: "Reviewing PR"},
			{State: "running", PaneTitle: "Running tests"},
		},
	}

	result := formatter.FormatWindow(window)
	expected := `0: dev (3 panes) | waiting: "Reviewing PR", running: "Building project", running: "Running tests"`
	if result != expected {
		t.Errorf("FormatWindow() = %q, want %q", result, expected)
	}
}

func TestFormatter_FormatWindow_NoSignals(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	window := tmux.WindowInfo{
		WindowIndex: 1,
		WindowName:  "shell",
		PaneCount:   2,
		Signals:     nil,
	}

	result := formatter.FormatWindow(window)
	expected := "1: shell (2 panes)"
	if result != expected {
		t.Errorf("FormatWindow() = %q, want %q", result, expected)
	}
}

func TestFormatter_FormatSession(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	session := tmux.SessionInfo{
		SessionName: "work",
		WindowCount: 5,
		Signals: []tmux.SignalInfo{
			{State: "running", PaneTitle: "Building"},
			{State: "waiting", PaneTitle: "Reviewing PR"},
			{State: "waiting", PaneTitle: "User input"},
		},
	}

	result := formatter.FormatSession(session)
	expected := `work: 5 windows | waiting: "Reviewing PR", waiting: "User input", running: "Building"`
	if result != expected {
		t.Errorf("FormatSession() = %q, want %q", result, expected)
	}
}

func TestFormatter_FormatSession_NoSignals(t *testing.T) {
	scheme := NewColorScheme(false)
	formatter := NewFormatter(scheme)

	session := tmux.SessionInfo{
		SessionName: "personal",
		WindowCount: 2,
		Signals:     nil,
	}

	result := formatter.FormatSession(session)
	expected := "personal: 2 windows"
	if result != expected {
		t.Errorf("FormatSession() = %q, want %q", result, expected)
	}
}

func TestFormatter_FormatWindow_WithColor(t *testing.T) {
	scheme := NewColorScheme(true)
	formatter := NewFormatter(scheme)

	window := tmux.WindowInfo{
		WindowIndex: 0,
		WindowName:  "dev",
		PaneCount:   2,
		Signals: []tmux.SignalInfo{
			{State: "waiting", PaneTitle: "Review"},
			{State: "running", PaneTitle: "Build"},
		},
	}

	result := formatter.FormatWindow(window)
	// Should contain color codes
	if result == `0: dev (2 panes) | waiting: "Review", running: "Build"` {
		t.Error("FormatWindow() should include color codes when color is enabled")
	}
	// Verify it contains the escape codes
	if len(result) <= len(`0: dev (2 panes) | waiting: "Review", running: "Build"`) {
		t.Error("FormatWindow() with color should be longer due to escape codes")
	}
}
