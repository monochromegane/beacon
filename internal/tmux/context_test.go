package tmux

import (
	"errors"
	"os"
	"testing"
)

type mockExecutor struct {
	output []byte
	err    error
}

func (m *mockExecutor) Execute(name string, args ...string) ([]byte, error) {
	return m.output, m.err
}

func TestContextProvider_GetEnvironment_NotInTmux(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	os.Unsetenv("TMUX")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		}
	}()

	provider := NewContextProvider()
	_, err := provider.GetEnvironment()
	if !errors.Is(err, ErrNotInTmux) {
		t.Errorf("GetEnvironment() error = %v, want ErrNotInTmux", err)
	}
}

func TestContextProvider_GetEnvironment_Success(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		} else {
			os.Unsetenv("TMUX")
		}
	}()

	executor := &mockExecutor{
		output: []byte("main\t0\t1\t%2\t⠋ Running tests\n"),
	}
	provider := NewContextProviderWithExecutor(executor)

	env, err := provider.GetEnvironment()
	if err != nil {
		t.Fatalf("GetEnvironment() error = %v", err)
	}

	if env.Type != "tmux" {
		t.Errorf("Type = %q, want %q", env.Type, "tmux")
	}
	if env.SessionName != "main" {
		t.Errorf("SessionName = %q, want %q", env.SessionName, "main")
	}
	if env.WindowIndex != 0 {
		t.Errorf("WindowIndex = %d, want %d", env.WindowIndex, 0)
	}
	if env.PaneIndex != 1 {
		t.Errorf("PaneIndex = %d, want %d", env.PaneIndex, 1)
	}
	if env.PaneID != "%2" {
		t.Errorf("PaneID = %q, want %q", env.PaneID, "%2")
	}
	if env.PaneTitle != "Running tests" {
		t.Errorf("PaneTitle = %q, want %q", env.PaneTitle, "Running tests")
	}
}

func TestContextProvider_GetEnvironment_EmptyPaneTitle(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		} else {
			os.Unsetenv("TMUX")
		}
	}()

	executor := &mockExecutor{
		output: []byte("main\t0\t1\t%2\t\n"),
	}
	provider := NewContextProviderWithExecutor(executor)

	env, err := provider.GetEnvironment()
	if err != nil {
		t.Fatalf("GetEnvironment() error = %v", err)
	}

	if env.PaneTitle != "" {
		t.Errorf("PaneTitle = %q, want empty string", env.PaneTitle)
	}
}

func TestContextProvider_GetEnvironment_CommandError(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		} else {
			os.Unsetenv("TMUX")
		}
	}()

	executor := &mockExecutor{
		err: errors.New("command failed"),
	}
	provider := NewContextProviderWithExecutor(executor)

	_, err := provider.GetEnvironment()
	if err == nil {
		t.Error("GetEnvironment() expected error, got nil")
	}
}

func TestExtractPaneTitleSummary(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"⠋ Running tests", "Running tests"},
		{"⠙ Compiling", "Compiling"},
		{"NoSpinner", "NoSpinner"},
		{"", ""},
		{"⠹ Multiple words here", "Multiple words here"},
	}

	for _, tt := range tests {
		got := extractPaneTitleSummary(tt.input)
		if got != tt.want {
			t.Errorf("extractPaneTitleSummary(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
