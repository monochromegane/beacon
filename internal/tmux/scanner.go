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
	PaneCount   int          `json:"pane_count"`
	Signals     []SignalInfo `json:"signals"`
}

// SessionInfo represents a tmux session with aggregated signals.
type SessionInfo struct {
	SessionName string       `json:"session_name"`
	WindowCount int          `json:"window_count"`
	Signals     []SignalInfo `json:"signals"`
}

// Scanner scans tmux windows and sessions for signals.
type Scanner struct {
	executor        CommandExecutor
	contextProvider *ContextProvider
}

// NewScanner creates a new Scanner with the default executor.
func NewScanner() *Scanner {
	executor := &DefaultExecutor{}
	return &Scanner{
		executor:        executor,
		contextProvider: NewContextProviderWithExecutor(executor),
	}
}

// NewScannerWithExecutor creates a new Scanner with a custom executor (for testing).
func NewScannerWithExecutor(executor CommandExecutor) *Scanner {
	return &Scanner{
		executor:        executor,
		contextProvider: NewContextProviderWithExecutor(executor),
	}
}

// ScanWindows scans all windows in the current session and matches signals.
func (s *Scanner) ScanWindows(signals []*signal.Signal) ([]WindowInfo, error) {
	output, err := s.executor.Execute("tmux", "list-windows", "-F", "#{window_index}\t#{window_name}\t#{window_id}\t#{window_panes}")
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
	for sessionName := range strings.SplitSeq(strings.TrimSpace(string(sessionOutput)), "\n") {
		if sessionName == "" {
			continue
		}
		output, err := s.executor.Execute("tmux", "list-windows", "-t", sessionName, "-F", "#{window_index}\t#{window_name}\t#{window_id}\t#{window_panes}")
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
		if len(parts) < 4 {
			continue
		}

		windowIndex, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		paneCount, err := strconv.Atoi(parts[3])
		if err != nil {
			paneCount = 1
		}

		window := WindowInfo{
			SessionName: sessionName,
			WindowIndex: windowIndex,
			WindowName:  parts[1],
			WindowID:    parts[2],
			PaneCount:   paneCount,
			Signals:     s.matchSignalsToWindow(signals, sessionName, windowIndex),
		}

		windows = append(windows, window)
	}

	return windows, nil
}

// matchSignalsToWindow finds all signals that belong to a specific tmux window.
func (s *Scanner) matchSignalsToWindow(signals []*signal.Signal, sessionName string, windowIndex int) []SignalInfo {
	var matched []SignalInfo
	for _, sig := range signals {
		env := sig.Environment
		if env == nil || env.Type != "tmux" {
			continue
		}
		if env.SessionName != sessionName || env.WindowIndex != windowIndex {
			continue
		}
		paneTitle := env.PaneTitle
		if env.PaneID != "" {
			if title, err := s.contextProvider.GetPaneTitle(env.PaneID); err == nil {
				paneTitle = title
			}
		}
		matched = append(matched, SignalInfo{
			SessionID:     sig.SessionID,
			State:         string(sig.State),
			Message:       sig.Message,
			CustomMessage: sig.CustomMessage,
			PaneIndex:     env.PaneIndex,
			PaneID:        env.PaneID,
			PaneTitle:     paneTitle,
		})
	}
	return matched
}

// ScanSessionsAggregated scans all sessions and returns aggregated session info.
func (s *Scanner) ScanSessionsAggregated(signals []*signal.Signal) ([]SessionInfo, error) {
	sessionOutput, err := s.executor.Execute("tmux", "list-sessions", "-F", "#{session_name}\t#{session_windows}")
	if err != nil {
		return nil, err
	}

	var sessions []SessionInfo
	for line := range strings.SplitSeq(strings.TrimSpace(string(sessionOutput)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		sessionName := parts[0]
		windowCount, err := strconv.Atoi(parts[1])
		if err != nil {
			windowCount = 0
		}

		session := SessionInfo{
			SessionName: sessionName,
			WindowCount: windowCount,
			Signals:     s.matchSignalsToSession(signals, sessionName),
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// ScanCurrentSessionAggregated scans the current session and returns aggregated session info.
func (s *Scanner) ScanCurrentSessionAggregated(signals []*signal.Signal) ([]SessionInfo, error) {
	sessionOutput, err := s.executor.Execute("tmux", "display-message", "-p", "#{session_name}\t#{session_windows}")
	if err != nil {
		return nil, err
	}

	line := strings.TrimSpace(string(sessionOutput))
	parts := strings.Split(line, "\t")
	if len(parts) < 2 {
		return nil, nil
	}

	sessionName := parts[0]
	windowCount, err := strconv.Atoi(parts[1])
	if err != nil {
		windowCount = 0
	}

	session := SessionInfo{
		SessionName: sessionName,
		WindowCount: windowCount,
		Signals:     s.matchSignalsToSession(signals, sessionName),
	}
	return []SessionInfo{session}, nil
}

// matchSignalsToSession finds all signals that belong to a specific tmux session.
func (s *Scanner) matchSignalsToSession(signals []*signal.Signal, sessionName string) []SignalInfo {
	var matched []SignalInfo
	for _, sig := range signals {
		env := sig.Environment
		if env == nil || env.Type != "tmux" {
			continue
		}
		if env.SessionName != sessionName {
			continue
		}
		paneTitle := env.PaneTitle
		if env.PaneID != "" {
			if title, err := s.contextProvider.GetPaneTitle(env.PaneID); err == nil {
				paneTitle = title
			}
		}
		matched = append(matched, SignalInfo{
			SessionID:     sig.SessionID,
			State:         string(sig.State),
			Message:       sig.Message,
			CustomMessage: sig.CustomMessage,
			PaneIndex:     env.PaneIndex,
			PaneID:        env.PaneID,
			PaneTitle:     paneTitle,
		})
	}
	return matched
}
