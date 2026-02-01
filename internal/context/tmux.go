package context

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ErrNotInTmux is returned when tmux context is requested outside of a tmux session.
var ErrNotInTmux = errors.New("not running inside tmux")

// TmuxContext represents tmux session/window/pane information.
type TmuxContext struct {
	SessionName string `json:"session_name"`
	WindowIndex int    `json:"window_index"`
	PaneIndex   int    `json:"pane_index"`
	PaneID      string `json:"pane_id"`
	PaneTitle   string `json:"pane_title"`
}

// extractPaneTitleSummary removes the spinner prefix from pane title.
// Claude Code sets pane titles like "⠋ Running tests" where the first character is a spinner.
// This function extracts just the summary part after the spinner.
func extractPaneTitleSummary(paneTitle string) string {
	if paneTitle == "" {
		return ""
	}
	parts := strings.SplitN(paneTitle, " ", 2)
	if len(parts) < 2 {
		return paneTitle
	}
	return parts[1]
}

// Type returns the context type identifier.
func (c *TmuxContext) Type() string {
	return "tmux"
}

// ToJSON serializes the context to JSON.
func (c *TmuxContext) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// TmuxProvider obtains tmux context information.
type TmuxProvider struct {
	executor CommandExecutor
}

// CommandExecutor is an interface for executing shell commands.
type CommandExecutor interface {
	Execute(name string, args ...string) ([]byte, error)
}

// DefaultExecutor is the default command executor using os/exec.
type DefaultExecutor struct{}

// Execute runs the command and returns its output.
func (e *DefaultExecutor) Execute(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// NewTmuxProvider creates a new TmuxProvider with the default executor.
func NewTmuxProvider() *TmuxProvider {
	return &TmuxProvider{executor: &DefaultExecutor{}}
}

// NewTmuxProviderWithExecutor creates a new TmuxProvider with a custom executor (for testing).
func NewTmuxProviderWithExecutor(executor CommandExecutor) *TmuxProvider {
	return &TmuxProvider{executor: executor}
}

// GetContext retrieves the current tmux context.
func (p *TmuxProvider) GetContext() (Context, error) {
	if os.Getenv("TMUX") == "" {
		return nil, ErrNotInTmux
	}

	output, err := p.executor.Execute("tmux", "display-message", "-p", "#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_id}\t#{pane_title}")
	if err != nil {
		return nil, err
	}

	// Use TrimSuffix instead of TrimSpace to preserve trailing tab when pane_title is empty
	parts := strings.Split(strings.TrimSuffix(string(output), "\n"), "\t")
	if len(parts) < 5 {
		return nil, errors.New("unexpected tmux output format")
	}

	windowIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, errors.New("invalid window index")
	}

	paneIndex, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, errors.New("invalid pane index")
	}

	return &TmuxContext{
		SessionName: parts[0],
		WindowIndex: windowIndex,
		PaneIndex:   paneIndex,
		PaneID:      parts[3],
		PaneTitle:   extractPaneTitleSummary(parts[4]),
	}, nil
}
