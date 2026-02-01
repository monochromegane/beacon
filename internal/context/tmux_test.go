package context

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

func TestTmuxContext_Type(t *testing.T) {
	ctx := &TmuxContext{}
	if ctx.Type() != "tmux" {
		t.Errorf("Type() = %q, want %q", ctx.Type(), "tmux")
	}
}

func TestTmuxContext_ToJSON(t *testing.T) {
	ctx := &TmuxContext{
		SessionName: "main",
		WindowIndex: 0,
		PaneIndex:   1,
		PaneID:      "%2",
		PaneTitle:   "Running tests",
	}

	data, err := ctx.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	expected := `{"session_name":"main","window_index":0,"pane_index":1,"pane_id":"%2","pane_title":"Running tests"}`
	if string(data) != expected {
		t.Errorf("ToJSON() = %q, want %q", string(data), expected)
	}
}

func TestTmuxProvider_GetContext_NotInTmux(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	os.Unsetenv("TMUX")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		}
	}()

	provider := NewTmuxProvider()
	_, err := provider.GetContext()
	if !errors.Is(err, ErrNotInTmux) {
		t.Errorf("GetContext() error = %v, want ErrNotInTmux", err)
	}
}

func TestTmuxProvider_GetContext_Success(t *testing.T) {
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
	provider := NewTmuxProviderWithExecutor(executor)

	ctx, err := provider.GetContext()
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}

	tmuxCtx, ok := ctx.(*TmuxContext)
	if !ok {
		t.Fatalf("GetContext() returned wrong type")
	}

	if tmuxCtx.SessionName != "main" {
		t.Errorf("SessionName = %q, want %q", tmuxCtx.SessionName, "main")
	}
	if tmuxCtx.WindowIndex != 0 {
		t.Errorf("WindowIndex = %d, want %d", tmuxCtx.WindowIndex, 0)
	}
	if tmuxCtx.PaneIndex != 1 {
		t.Errorf("PaneIndex = %d, want %d", tmuxCtx.PaneIndex, 1)
	}
	if tmuxCtx.PaneID != "%2" {
		t.Errorf("PaneID = %q, want %q", tmuxCtx.PaneID, "%2")
	}
	if tmuxCtx.PaneTitle != "Running tests" {
		t.Errorf("PaneTitle = %q, want %q", tmuxCtx.PaneTitle, "Running tests")
	}
}

func TestTmuxProvider_GetContext_CommandError(t *testing.T) {
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
	provider := NewTmuxProviderWithExecutor(executor)

	_, err := provider.GetContext()
	if err == nil {
		t.Error("GetContext() expected error, got nil")
	}
}

func TestExtractPaneTitleSummary(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"spinner with text", "⠋ Running tests", "Running tests"},
		{"spinner with multi-word text", "⠙ Exploring codebase structure", "Exploring codebase structure"},
		{"no space separator", "NoSpace", "NoSpace"},
		{"only spinner", "⠋", "⠋"},
		{"multiple spaces", "⠋ Multiple   spaces  here", "Multiple   spaces  here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPaneTitleSummary(tt.input)
			if result != tt.expected {
				t.Errorf("extractPaneTitleSummary(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTmuxProvider_GetContext_EmptyPaneTitle(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		} else {
			os.Unsetenv("TMUX")
		}
	}()

	// When pane_title is empty, tmux outputs an empty string for that field.
	// The output is "main\t0\t1\t%2\t\n" (with trailing tab and newline but empty title).
	executor := &mockExecutor{
		output: []byte("main\t0\t1\t%2\t\n"),
	}
	provider := NewTmuxProviderWithExecutor(executor)

	ctx, err := provider.GetContext()
	if err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}

	tmuxCtx, ok := ctx.(*TmuxContext)
	if !ok {
		t.Fatalf("GetContext() returned wrong type")
	}

	if tmuxCtx.PaneTitle != "" {
		t.Errorf("PaneTitle = %q, want empty string", tmuxCtx.PaneTitle)
	}
}

func TestTmuxProvider_GetContext_InvalidOutput(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		} else {
			os.Unsetenv("TMUX")
		}
	}()

	tests := []struct {
		name   string
		output string
	}{
		{"too few parts", "main\t0\t1\t%2"},
		{"invalid window index", "main\tabc\t1\t%2\t\n"},
		{"invalid pane index", "main\t0\tabc\t%2\t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{
				output: []byte(tt.output),
			}
			provider := NewTmuxProviderWithExecutor(executor)

			_, err := provider.GetContext()
			if err == nil {
				t.Error("GetContext() expected error, got nil")
			}
		})
	}
}
