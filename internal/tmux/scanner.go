package tmux

import (
	"strconv"
	"strings"

	"github.com/monochromegane/beacon/internal/signal"
)

// SignalInfo represents signal information for a pane within a window.
type SignalInfo struct {
	SessionID     string `json:"session_id"`
	State         string `json:"state"`
	Message       string `json:"message"`
	CustomMessage string `json:"custom_message,omitempty"`
	PaneIndex     int    `json:"pane_index"`
	PaneID        string `json:"pane_id"`
	PaneTitle     string `json:"pane_title"`
}

// WindowInfo represents a tmux window with its signals.
type WindowInfo struct {
	SessionName string       `json:"session_name"`
	WindowIndex int          `json:"window_index"`
	WindowName  string       `json:"window_name"`
	WindowID    string       `json:"window_id"`
	Signals     []SignalInfo `json:"signals"`
}

// Scanner scans tmux windows and sessions for signals.
type Scanner struct {
	executor CommandExecutor
}

// NewScanner creates a new Scanner with the default executor.
func NewScanner() *Scanner {
	return &Scanner{executor: &DefaultExecutor{}}
}

// NewScannerWithExecutor creates a new Scanner with a custom executor (for testing).
func NewScannerWithExecutor(executor CommandExecutor) *Scanner {
	return &Scanner{executor: executor}
}

// ScanWindows scans all windows in the current session and matches signals.
func (s *Scanner) ScanWindows(signals []*signal.Signal) ([]WindowInfo, error) {
	output, err := s.executor.Execute("tmux", "list-windows", "-F", "#{window_index}\t#{window_name}\t#{window_id}")
	if err != nil {
		return nil, err
	}

	sessionOutput, err := s.executor.Execute("tmux", "display-message", "-p", "#{session_name}")
	if err != nil {
		return nil, err
	}
	sessionName := strings.TrimSpace(string(sessionOutput))

	return s.parseWindowsAndMatchSignals(sessionName, string(output), signals)
}

// ScanSessions scans all sessions and their windows and matches signals.
func (s *Scanner) ScanSessions(signals []*signal.Signal) ([]WindowInfo, error) {
	sessionOutput, err := s.executor.Execute("tmux", "list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil, err
	}

	var allWindows []WindowInfo
	sessions := strings.Split(strings.TrimSpace(string(sessionOutput)), "\n")
	for _, sessionName := range sessions {
		if sessionName == "" {
			continue
		}
		output, err := s.executor.Execute("tmux", "list-windows", "-t", sessionName, "-F", "#{window_index}\t#{window_name}\t#{window_id}")
		if err != nil {
			continue
		}
		windows, err := s.parseWindowsAndMatchSignals(sessionName, string(output), signals)
		if err != nil {
			continue
		}
		allWindows = append(allWindows, windows...)
	}
	return allWindows, nil
}

// parseWindowsAndMatchSignals parses tmux window output and matches signals to windows.
func (s *Scanner) parseWindowsAndMatchSignals(sessionName, output string, signals []*signal.Signal) ([]WindowInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var windows []WindowInfo

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		windowIndex, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		window := WindowInfo{
			SessionName: sessionName,
			WindowIndex: windowIndex,
			WindowName:  parts[1],
			WindowID:    parts[2],
		}

		// Match signals to this window based on environment
		for _, sig := range signals {
			if sig.Environment != nil &&
				sig.Environment.Type == "tmux" &&
				sig.Environment.SessionName == sessionName &&
				sig.Environment.WindowIndex == windowIndex {
				window.Signals = append(window.Signals, SignalInfo{
					SessionID:     sig.SessionID,
					State:         string(sig.State),
					Message:       sig.Message,
					CustomMessage: sig.CustomMessage,
					PaneIndex:     sig.Environment.PaneIndex,
					PaneID:        sig.Environment.PaneID,
					PaneTitle:     sig.Environment.PaneTitle,
				})
			}
		}

		windows = append(windows, window)
	}

	return windows, nil
}
