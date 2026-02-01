package signal

import (
	"testing"
	"time"
)

func TestSignalStates(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateStarted, "started"},
		{StateRunning, "running"},
		{StateWaiting, "waiting"},
		{StateIdle, "idle"},
	}

	for _, tt := range tests {
		if string(tt.state) != tt.want {
			t.Errorf("State = %q, want %q", tt.state, tt.want)
		}
	}
}

func TestSignalFields(t *testing.T) {
	now := time.Now()
	env := &Environment{
		Type:        "tmux",
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
		PaneTitle:   "Running tests",
	}

	sig := &Signal{
		SessionID:     "test-session",
		SignalType:    "claude",
		State:         StateRunning,
		Message:       "claude:running",
		CustomMessage: "custom",
		Source:        "cli",
		UpdatedAt:     now,
		Environment:   env,
	}

	if sig.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", sig.SessionID, "test-session")
	}
	if sig.SignalType != "claude" {
		t.Errorf("SignalType = %q, want %q", sig.SignalType, "claude")
	}
	if sig.State != StateRunning {
		t.Errorf("State = %q, want %q", sig.State, StateRunning)
	}
	if sig.Message != "claude:running" {
		t.Errorf("Message = %q, want %q", sig.Message, "claude:running")
	}
	if sig.CustomMessage != "custom" {
		t.Errorf("CustomMessage = %q, want %q", sig.CustomMessage, "custom")
	}
	if sig.Source != "cli" {
		t.Errorf("Source = %q, want %q", sig.Source, "cli")
	}
	if sig.UpdatedAt != now {
		t.Errorf("UpdatedAt = %v, want %v", sig.UpdatedAt, now)
	}
	if sig.Environment.Type != "tmux" {
		t.Errorf("Environment.Type = %q, want %q", sig.Environment.Type, "tmux")
	}
}
