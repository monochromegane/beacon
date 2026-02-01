package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/alecthomas/kong"
	"github.com/monochromegane/beacon/internal/signal"
	"github.com/monochromegane/beacon/internal/tmux"
)

const cmdName = "beacon"

// EmitCmd emits a beacon signal based on hook events from stdin.
type EmitCmd struct {
	Signal  string `name:"signal" short:"S" help:"Signal type" default:"claude" enum:"claude"`
	Env     string `name:"env" short:"E" help:"Environment type" default:"tmux" enum:"tmux,"`
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

	// If SessionEnd, delete signal and exit
	if result.ShouldDelete {
		store, err := cli.getSignalStore()
		if err != nil {
			return err
		}
		return store.Delete(c.Signal, event.SessionID)
	}

	// Get environment if specified
	var env *signal.Environment
	if c.Env == "tmux" {
		provider := cli.getTmuxContextProvider()
		env, err = provider.GetEnvironment()
		if err != nil && !errors.Is(err, tmux.ErrNotInTmux) {
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
	store, err := cli.getSignalStore()
	if err != nil {
		return err
	}
	return store.Write(sig)
}

// ScanCmd scans tmux windows/sessions for signals.
type ScanCmd struct {
	Signal   string `name:"signal" short:"S" help:"Signal type" default:"claude" enum:"claude"`
	Env      string `name:"env" short:"E" help:"Environment type" default:"tmux" enum:"tmux"`
	Scope    string `name:"scope" short:"s" help:"Scan scope" default:"window" enum:"window,session"`
	Template string `name:"template" short:"t" help:"Go template for output"`
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

	scanner := cli.getTmuxScanner()
	var windows []tmux.WindowInfo

	switch c.Scope {
	case "window":
		windows, err = scanner.ScanWindows(signals)
	case "session":
		windows, err = scanner.ScanSessions(signals)
	}
	if err != nil {
		return err
	}

	// Output
	if c.Template != "" {
		return c.outputWithTemplate(cli.out, windows)
	}
	return c.outputDefault(cli.out, windows)
}

// outputDefault outputs in default format: {window_index}:{window_name}:{comma_separated_states}
func (c *ScanCmd) outputDefault(out io.Writer, windows []tmux.WindowInfo) error {
	for _, w := range windows {
		states := make([]string, len(w.Signals))
		for i, s := range w.Signals {
			states[i] = s.State
		}
		fmt.Fprintf(out, "%d:%s:%s\n", w.WindowIndex, w.WindowName, strings.Join(states, ","))
	}
	return nil
}

// outputWithTemplate outputs using a Go template.
func (c *ScanCmd) outputWithTemplate(out io.Writer, windows []tmux.WindowInfo) error {
	tmpl, err := template.New("scan").Parse(c.Template)
	if err != nil {
		return err
	}

	for _, w := range windows {
		// Convert window to map for template
		data := map[string]any{
			"SessionName": w.SessionName,
			"WindowIndex": w.WindowIndex,
			"WindowName":  w.WindowName,
			"WindowID":    w.WindowID,
			"Signals":     w.Signals,
		}
		// Also provide JSON of signals for advanced processing
		signalsJSON, _ := json.Marshal(w.Signals)
		data["SignalsJSON"] = string(signalsJSON)

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return err
		}
		// Only output if template produced non-empty result
		if result := strings.TrimSpace(buf.String()); result != "" {
			fmt.Fprintln(out, result)
		}
	}
	return nil
}

// CLI represents the beacon command-line interface.
type CLI struct {
	Version kong.VersionFlag `help:"Show version"`
	Emit    EmitCmd          `cmd:"" help:"Emit a beacon signal from hook event"`
	Scan    ScanCmd          `cmd:"" help:"Scan for beacon signals in tmux"`

	signalStore         signal.Store
	tmuxContextProvider *tmux.ContextProvider
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

func (c *CLI) getTmuxContextProvider() *tmux.ContextProvider {
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
