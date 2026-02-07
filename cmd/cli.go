package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/alecthomas/kong"
	"github.com/monochromegane/beacon/internal/output"
	"github.com/monochromegane/beacon/internal/signal"
	"github.com/monochromegane/beacon/internal/tmux"
)

const cmdName = "beacon"

// EnvironmentProvider is an interface for obtaining environment information.
type EnvironmentProvider interface {
	GetEnvironment() (*signal.Environment, error)
	GetPaneTitle(paneID string) (string, error)
	ListPaneIDs() ([]string, error)
}

// EmitCmd emits a beacon signal based on hook events from stdin.
type EmitCmd struct {
	Signal  string `name:"signal" short:"S" help:"Signal type" default:"claude" enum:"claude"`
	Env     string `name:"env" short:"E" help:"Environment type" default:"tmux" enum:"tmux,none"`
	Message string `arg:"" optional:"" help:"Optional custom message"`
}

func (c *EmitCmd) Run(cli *CLI) error {
	// Parse hook event from stdin
	event, err := signal.ParseHookEvent(cli.in)
	if err != nil {
		return fmt.Errorf("failed to parse hook event: %w", err)
	}

	// Map event to state
	result := signal.MapEventToState(event)

	// Get store early (used for both delete and environment preservation)
	store, err := cli.getSignalStore()
	if err != nil {
		return err
	}

	// If SessionEnd, delete signal and exit
	if result.ShouldDelete {
		return store.Delete(c.Signal, event.SessionID)
	}

	// If ShouldSkip, do not update signal and exit successfully
	if result.ShouldSkip {
		return nil
	}

	// Get environment if specified (preserving existing window/pane info)
	var env *signal.Environment
	if c.Env == "tmux" {
		env, err = c.getPreservedEnvironment(cli, store, event.SessionID)
		if err != nil {
			return err
		}
	}

	// Create signal
	sig := &signal.Signal{
		SessionID:     event.SessionID,
		SignalType:    c.Signal,
		State:         result.State,
		Message:       result.Message,
		CustomMessage: c.Message,
		Source:        event.Source,
		UpdatedAt:     time.Now(),
		Environment:   env,
	}

	// Save signal
	return store.Write(sig)
}

// getPreservedEnvironment returns an Environment that preserves existing window/pane info.
// If an existing signal exists, it reuses the Environment (only updating PaneTitle).
// If no existing signal exists, it fetches fresh environment from tmux.
func (c *EmitCmd) getPreservedEnvironment(cli *CLI, store signal.Store, sessionID string) (*signal.Environment, error) {
	provider := cli.getTmuxContextProvider()

	existingSignal, err := store.Read(c.Signal, sessionID)
	if err == nil && existingSignal != nil && existingSignal.Environment != nil {
		// Copy and reuse existing Environment
		env := *existingSignal.Environment

		// Update only PaneTitle with current value from the correct pane
		paneTitle, err := provider.GetPaneTitle(existingSignal.Environment.PaneID)
		if err == nil {
			env.PaneTitle = paneTitle
		}

		return &env, nil
	}

	// No existing signal, fetch fresh environment
	env, err := provider.GetEnvironment()
	if err != nil && !errors.Is(err, tmux.ErrNotInTmux) {
		return nil, err
	}
	return env, nil
}

// CleanCmd removes stale beacon signals whose tmux panes no longer exist.
type CleanCmd struct {
	Signal string `name:"signal" short:"S" help:"Signal type" default:"claude" enum:"claude"`
	Env    string `name:"env" short:"E" help:"Environment type" default:"tmux" enum:"tmux,none"`
}

func (c *CleanCmd) Run(cli *CLI) error {
	if c.Env == "none" {
		return nil
	}

	store, err := cli.getSignalStore()
	if err != nil {
		return err
	}

	signals, err := store.List(c.Signal)
	if err != nil {
		return err
	}

	provider := cli.getTmuxContextProvider()

	paneIDs, err := provider.ListPaneIDs()
	if err != nil {
		return err
	}
	activePane := make(map[string]bool, len(paneIDs))
	for _, id := range paneIDs {
		activePane[id] = true
	}

	// Track newest signal per pane to detect duplicates
	newestByPane := make(map[string]*signal.Signal)
	for _, sig := range signals {
		if sig.Environment == nil || sig.Environment.Type != "tmux" || sig.Environment.PaneID == "" {
			continue
		}
		if existing, ok := newestByPane[sig.Environment.PaneID]; !ok || sig.UpdatedAt.After(existing.UpdatedAt) {
			newestByPane[sig.Environment.PaneID] = sig
		}
	}

	for _, sig := range signals {
		if sig.Environment == nil || sig.Environment.Type != "tmux" || sig.Environment.PaneID == "" {
			continue
		}

		// Delete if pane no longer exists, or if a newer signal owns the same pane
		if !activePane[sig.Environment.PaneID] || newestByPane[sig.Environment.PaneID].SessionID != sig.SessionID {
			if delErr := store.Delete(sig.SignalType, sig.SessionID); delErr != nil {
				return delErr
			}
		}
	}

	return nil
}

// ScanCmd scans tmux windows/sessions for signals.
type ScanCmd struct {
	Signal      string `name:"signal" short:"S" help:"Signal type" default:"claude" enum:"claude"`
	Env         string `name:"env" short:"E" help:"Environment type" default:"tmux" enum:"tmux,none"`
	Scope       string `name:"scope" short:"s" help:"Scan scope: window or session" default:"window" enum:"window,session"`
	AllSessions bool   `name:"all-sessions" short:"a" help:"Scan all sessions instead of current session only"`
	Template    string `name:"template" short:"t" help:"Go template for output"`
	Color       string `name:"color" help:"Color output: always, auto, never" default:"auto" enum:"always,auto,never"`
}

func (c *ScanCmd) Run(cli *CLI) error {
	store, err := cli.getSignalStore()
	if err != nil {
		return err
	}

	signals, err := store.List(c.Signal)
	if err != nil {
		return err
	}

	if c.Env == "none" {
		return c.runWithoutEnv(cli, signals)
	}
	return c.runWithTmux(cli, signals)
}

// runWithoutEnv outputs signals without tmux dependency.
func (c *ScanCmd) runWithoutEnv(cli *CLI, signals []*signal.Signal) error {
	views := make([]signal.View, len(signals))
	for i, sig := range signals {
		views[i] = sig.ToView()
	}

	if c.Template != "" {
		return c.outputWithTemplate(cli.out, map[string]any{
			"Signals": views,
		})
	}

	useColor := output.ShouldUseColor(c.Color)
	scheme := output.NewColorScheme(useColor)
	formatter := output.NewFormatter(scheme)

	sorted := output.SortByPriority(views)
	for _, sig := range sorted {
		fmt.Fprintln(cli.out, formatter.FormatSignal(sig))
	}
	return nil
}

// runWithTmux outputs signals using tmux scanner.
func (c *ScanCmd) runWithTmux(cli *CLI, signals []*signal.Signal) error {
	scanner := cli.getTmuxScanner()

	switch c.Scope {
	case "window":
		if c.AllSessions {
			windows, err := scanner.ScanSessions(signals)
			if err != nil {
				return err
			}
			if c.Template != "" {
				return c.outputWindowsWithTemplate(cli.out, windows)
			}
			return c.outputAllWindowsDefault(cli.out, windows)
		}
		windows, err := scanner.ScanWindows(signals)
		if err != nil {
			return err
		}
		if c.Template != "" {
			return c.outputWindowsWithTemplate(cli.out, windows)
		}
		return c.outputWindowsDefault(cli.out, windows)
	case "session":
		if c.AllSessions {
			sessions, err := scanner.ScanSessionsAggregated(signals)
			if err != nil {
				return err
			}
			if c.Template != "" {
				return c.outputSessionsWithTemplate(cli.out, sessions)
			}
			return c.outputSessionsDefault(cli.out, sessions)
		}
		sessions, err := scanner.ScanCurrentSessionAggregated(signals)
		if err != nil {
			return err
		}
		if c.Template != "" {
			return c.outputSessionsWithTemplate(cli.out, sessions)
		}
		return c.outputSessionsDefault(cli.out, sessions)
	}
	return nil
}

// outputWindowsDefault outputs windows in the new format with priority sorting and colors.
func (c *ScanCmd) outputWindowsDefault(out io.Writer, windows []tmux.WindowInfo) error {
	useColor := output.ShouldUseColor(c.Color)
	scheme := output.NewColorScheme(useColor)
	formatter := output.NewFormatter(scheme)

	for _, w := range windows {
		fmt.Fprintln(out, formatter.FormatWindow(w))
	}
	return nil
}

// outputSessionsDefault outputs sessions in the new format with priority sorting and colors.
func (c *ScanCmd) outputSessionsDefault(out io.Writer, sessions []tmux.SessionInfo) error {
	useColor := output.ShouldUseColor(c.Color)
	scheme := output.NewColorScheme(useColor)
	formatter := output.NewFormatter(scheme)

	for _, s := range sessions {
		fmt.Fprintln(out, formatter.FormatSession(s))
	}
	return nil
}

// outputAllWindowsDefault outputs all windows from all sessions with session name prefix.
func (c *ScanCmd) outputAllWindowsDefault(out io.Writer, windows []tmux.WindowInfo) error {
	useColor := output.ShouldUseColor(c.Color)
	scheme := output.NewColorScheme(useColor)
	formatter := output.NewFormatter(scheme)

	for _, w := range windows {
		fmt.Fprintln(out, formatter.FormatWindowWithSession(w))
	}
	return nil
}

// outputWithTemplate outputs data using a Go template.
func (c *ScanCmd) outputWithTemplate(out io.Writer, data map[string]any) error {
	tmpl, err := template.New("scan").Parse(c.Template)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	// Only output if template produced non-empty result
	if result := strings.TrimSpace(buf.String()); result != "" {
		fmt.Fprintln(out, result)
	}
	return nil
}

// flattenWindowSignals collects all signals from windows into a flat slice.
func flattenWindowSignals(windows []tmux.WindowInfo) []signal.View {
	var views []signal.View
	for _, w := range windows {
		views = append(views, w.Signals...)
	}
	return views
}

// flattenSessionSignals collects all signals from sessions into a flat slice.
func flattenSessionSignals(sessions []tmux.SessionInfo) []signal.View {
	var views []signal.View
	for _, s := range sessions {
		views = append(views, s.Signals...)
	}
	return views
}

// outputWindowsWithTemplate outputs windows using a Go template.
func (c *ScanCmd) outputWindowsWithTemplate(out io.Writer, windows []tmux.WindowInfo) error {
	return c.outputWithTemplate(out, map[string]any{
		"Windows": windows,
		"Signals": flattenWindowSignals(windows),
	})
}

// outputSessionsWithTemplate outputs sessions using a Go template.
func (c *ScanCmd) outputSessionsWithTemplate(out io.Writer, sessions []tmux.SessionInfo) error {
	return c.outputWithTemplate(out, map[string]any{
		"Sessions": sessions,
		"Signals":  flattenSessionSignals(sessions),
	})
}

// CLI represents the beacon command-line interface.
type CLI struct {
	Version kong.VersionFlag `help:"Show version"`
	Emit    EmitCmd          `cmd:"" help:"Emit a beacon signal from hook event"`
	Scan    ScanCmd          `cmd:"" help:"Scan for beacon signals in tmux"`
	Clean   CleanCmd         `cmd:"" help:"Remove stale beacon signals"`

	signalStore         signal.Store
	tmuxContextProvider EnvironmentProvider
	tmuxScanner         *tmux.Scanner
	in                  io.Reader
	out                 io.Writer
}

// NewCLI creates a new CLI instance.
func NewCLI() *CLI {
	return &CLI{}
}

func (c *CLI) getSignalStore() (signal.Store, error) {
	if c.signalStore == nil {
		store, err := signal.NewFileStore()
		if err != nil {
			return nil, err
		}
		c.signalStore = store
	}
	return c.signalStore, nil
}

func (c *CLI) getTmuxContextProvider() EnvironmentProvider {
	if c.tmuxContextProvider == nil {
		c.tmuxContextProvider = tmux.NewContextProvider()
	}
	return c.tmuxContextProvider
}

func (c *CLI) getTmuxScanner() *tmux.Scanner {
	if c.tmuxScanner == nil {
		c.tmuxScanner = tmux.NewScanner()
	}
	return c.tmuxScanner
}

func (c *CLI) initDefaults() {
	if c.in == nil {
		c.in = os.Stdin
	}
	if c.out == nil {
		c.out = os.Stdout
	}
}

// Execute runs the CLI with the given arguments.
func (c *CLI) Execute(args []string) error {
	c.initDefaults()

	parser, err := kong.New(c,
		kong.Name(cmdName),
		kong.Description("A CLI tool for managing coding agent states"),
		kong.UsageOnError(),
		kong.Vars{
			"version": fmt.Sprintf("%s v%s (rev:%s)", cmdName, version, revision),
		},
		kong.Bind(c),
	)
	if err != nil {
		return err
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	return ctx.Run(c)
}
