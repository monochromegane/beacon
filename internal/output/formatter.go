package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/monochromegane/beacon/internal/tmux"
)

// State priority for sorting (lower number = higher priority)
var statePriority = map[string]int{
	"waiting": 1,
	"idle":    2,
	"started": 2,
	"running": 3,
}

// Formatter handles formatting of scan output with optional color support.
type Formatter struct {
	scheme ColorScheme
}

// NewFormatter creates a new Formatter with the given color scheme.
func NewFormatter(scheme ColorScheme) *Formatter {
	return &Formatter{scheme: scheme}
}

// SortByPriority sorts signals by state priority (waiting > idle/started > running).
func SortByPriority(signals []tmux.SignalInfo) []tmux.SignalInfo {
	sorted := make([]tmux.SignalInfo, len(signals))
	copy(sorted, signals)
	sort.SliceStable(sorted, func(i, j int) bool {
		pi := statePriority[sorted[i].State]
		pj := statePriority[sorted[j].State]
		if pi == 0 {
			pi = 99 // unknown states go last
		}
		if pj == 0 {
			pj = 99
		}
		return pi < pj
	})
	return sorted
}

// ColorizeState applies color to a state string.
func (f *Formatter) ColorizeState(state string) string {
	var color string
	switch state {
	case "waiting":
		color = f.scheme.Waiting
	case "idle":
		color = f.scheme.Idle
	case "started":
		color = f.scheme.Started
	case "running":
		color = f.scheme.Running
	}
	if color == "" {
		return state
	}
	return color + state + f.scheme.Reset
}

// FormatSignals formats signals as 'state: "title", ...' sorted by priority.
func (f *Formatter) FormatSignals(signals []tmux.SignalInfo) string {
	if len(signals) == 0 {
		return ""
	}

	sorted := SortByPriority(signals)
	parts := make([]string, len(sorted))
	for i, sig := range sorted {
		title := sig.PaneTitle
		if title == "" {
			title = sig.CustomMessage
		}
		coloredState := f.ColorizeState(sig.State)
		if title != "" {
			parts[i] = fmt.Sprintf("%s: %q", coloredState, title)
		} else {
			parts[i] = coloredState
		}
	}
	return strings.Join(parts, ", ")
}

// FormatWindow formats a single window line.
func (f *Formatter) FormatWindow(w tmux.WindowInfo) string {
	signalsStr := f.FormatSignals(w.Signals)
	if signalsStr != "" {
		return fmt.Sprintf("%d: %s (%d panes) | %s", w.WindowIndex, w.WindowName, w.PaneCount, signalsStr)
	}
	return fmt.Sprintf("%d: %s (%d panes)", w.WindowIndex, w.WindowName, w.PaneCount)
}

// FormatSession formats a single session line.
func (f *Formatter) FormatSession(s tmux.SessionInfo) string {
	signalsStr := f.FormatSignals(s.Signals)
	if signalsStr != "" {
		return fmt.Sprintf("%s: %d windows | %s", s.SessionName, s.WindowCount, signalsStr)
	}
	return fmt.Sprintf("%s: %d windows", s.SessionName, s.WindowCount)
}

// FormatWindowWithSession formats a single window line with session name prefix.
// Output format: "{session}:{index}: {name} ({panes} panes) | {signals}"
func (f *Formatter) FormatWindowWithSession(w tmux.WindowInfo) string {
	signalsStr := f.FormatSignals(w.Signals)
	if signalsStr != "" {
		return fmt.Sprintf("%s:%d: %s (%d panes) | %s",
			w.SessionName, w.WindowIndex, w.WindowName, w.PaneCount, signalsStr)
	}
	return fmt.Sprintf("%s:%d: %s (%d panes)",
		w.SessionName, w.WindowIndex, w.WindowName, w.PaneCount)
}
