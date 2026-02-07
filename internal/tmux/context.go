package tmux

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/monochromegane/beacon/internal/signal"
)

// ErrNotInTmux is returned when tmux context is requested outside of a tmux session.
var ErrNotInTmux = errors.New("not running inside tmux")

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

// ContextProvider obtains tmux context information.
type ContextProvider struct {
	executor CommandExecutor
}

// NewContextProvider creates a new ContextProvider with the default executor.
func NewContextProvider() *ContextProvider {
	return &ContextProvider{executor: &DefaultExecutor{}}
}

// NewContextProviderWithExecutor creates a new ContextProvider with a custom executor (for testing).
func NewContextProviderWithExecutor(executor CommandExecutor) *ContextProvider {
	return &ContextProvider{executor: executor}
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

// GetEnvironment retrieves the current tmux environment.
func (p *ContextProvider) GetEnvironment() (*signal.Environment, error) {
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

	return &signal.Environment{
		Type:        "tmux",
		SessionName: parts[0],
		WindowIndex: windowIndex,
		PaneIndex:   paneIndex,
		PaneID:      parts[3],
		PaneTitle:   extractPaneTitleSummary(parts[4]),
	}, nil
}

// GetPaneTitle retrieves the pane title for a specific pane ID.
func (p *ContextProvider) GetPaneTitle(paneID string) (string, error) {
	output, err := p.executor.Execute("tmux", "display-message", "-t", paneID, "-p", "#{pane_title}")
	if err != nil {
		return "", err
	}
	return extractPaneTitleSummary(strings.TrimSuffix(string(output), "\n")), nil
}

// ListPaneIDs returns all pane IDs across all tmux sessions.
func (p *ContextProvider) ListPaneIDs() ([]string, error) {
	output, err := p.executor.Execute("tmux", "list-panes", "-a", "-F", "#{pane_id}")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	var ids []string
	for _, line := range lines {
		if id := strings.TrimSpace(line); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
