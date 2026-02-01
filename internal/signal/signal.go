package signal

import (
	"time"
)

// State represents the current state of a coding agent session.
type State string

const (
	StateStarted State = "started"
	StateRunning State = "running"
	StateWaiting State = "waiting"
	StateIdle    State = "idle"
)

// Signal represents a coding agent's current state and context.
type Signal struct {
	SessionID     string       `json:"session_id"`
	SignalType    string       `json:"signal_type"`
	State         State        `json:"state"`
	Message       string       `json:"message"`
	CustomMessage string       `json:"custom_message,omitempty"`
	Source        string       `json:"source,omitempty"`
	UpdatedAt     time.Time    `json:"updated_at"`
	Environment   *Environment `json:"environment,omitempty"`
}

// Environment represents the terminal environment context.
type Environment struct {
	Type        string `json:"type"`
	SessionName string `json:"session_name"`
	WindowIndex int    `json:"window_index"`
	PaneIndex   int    `json:"pane_index"`
	PaneID      string `json:"pane_id"`
	PaneTitle   string `json:"pane_title"`
}
