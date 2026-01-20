package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/alecthomas/kong"
	"github.com/monochromegane/beacon/internal/beacon"
	"github.com/monochromegane/beacon/internal/context"
)

const cmdName = "beacon"

type EmitCmd struct {
	Message string `arg:"" help:"Message to emit"`
	Context string `name:"context" short:"c" help:"Context type (tmux)" enum:",tmux" default:""`
}

func (c *EmitCmd) Run(cli *CLI) error {
	b, err := cli.newBeacon()
	if err != nil {
		return err
	}

	if c.Context == "" {
		return b.Emit(cli.ppid, c.Message)
	}

	ctx, err := cli.getContext(c.Context)
	if err != nil {
		return err
	}
	return b.EmitWithContext(cli.ppid, c.Message, ctx)
}

type SilenceCmd struct{}

func (c *SilenceCmd) Run(cli *CLI) error {
	b, err := cli.newBeacon()
	if err != nil {
		return err
	}
	return b.Silence(cli.ppid)
}

type ListCmd struct{}

func (c *ListCmd) Run(cli *CLI) error {
	b, err := cli.newBeacon()
	if err != nil {
		return err
	}
	return b.List()
}

type ContextCmd struct {
	PID      int    `arg:"" help:"Process ID to read context for"`
	Template string `name:"template" short:"t" help:"Go text/template string for custom formatting" default:""`
}

func (c *ContextCmd) Run(cli *CLI) error {
	store, err := cli.getContextStore()
	if err != nil {
		return err
	}

	data, err := store.Read(c.PID)
	if err != nil {
		return err
	}

	if c.Template == "" {
		cli.out.Write(data)
		cli.out.Write([]byte("\n"))
		return nil
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	tmpl, err := template.New("context").Parse(c.Template)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(cli.out, m); err != nil {
		return err
	}
	cli.out.Write([]byte("\n"))
	return nil
}

type CLI struct {
	Version kong.VersionFlag `help:"Show version"`
	Emit    EmitCmd          `cmd:"" help:"Emit a beacon signal"`
	Silence SilenceCmd       `cmd:"" help:"Silence the beacon"`
	List    ListCmd          `cmd:"" help:"List all active beacons"`
	Context ContextCmd       `cmd:"" help:"Display context for a process"`

	ppid         int
	store        beacon.Store
	contextStore context.ContextStore
	out          io.Writer
}

func NewCLI(ppid int) *CLI {
	return &CLI{
		ppid: ppid,
	}
}

func (c *CLI) newBeacon() (*beacon.Beacon, error) {
	if c.store == nil {
		store, err := beacon.NewFileStore()
		if err != nil {
			return nil, err
		}
		c.store = store
	}
	if c.contextStore == nil {
		contextStore, err := context.NewFileContextStore()
		if err != nil {
			return nil, err
		}
		c.contextStore = contextStore
	}
	if c.out == nil {
		c.out = os.Stdout
	}
	return beacon.NewWithContextStore(c.store, c.contextStore, c.out), nil
}

func (c *CLI) getContextStore() (context.ContextStore, error) {
	if c.contextStore == nil {
		contextStore, err := context.NewFileContextStore()
		if err != nil {
			return nil, err
		}
		c.contextStore = contextStore
	}
	if c.out == nil {
		c.out = os.Stdout
	}
	return c.contextStore, nil
}

func (c *CLI) getContext(contextType string) (context.Context, error) {
	switch contextType {
	case "tmux":
		provider := context.NewTmuxProvider()
		return provider.GetContext()
	default:
		return nil, errors.New("unknown context type: " + contextType)
	}
}

func (c *CLI) Execute(args []string) error {
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
