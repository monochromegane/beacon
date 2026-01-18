package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/monochromegane/beacon/internal/beacon"
)

const cmdName = "beacon"

type EmitCmd struct {
	Message string `arg:"" help:"Message to emit"`
}

func (c *EmitCmd) Run(cli *CLI) error {
	b, err := cli.newBeacon()
	if err != nil {
		return err
	}
	return b.Emit(cli.ppid, c.Message)
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

type CLI struct {
	Version kong.VersionFlag `help:"Show version"`
	Emit    EmitCmd          `cmd:"" help:"Emit a beacon signal"`
	Silence SilenceCmd       `cmd:"" help:"Silence the beacon"`
	List    ListCmd          `cmd:"" help:"List all active beacons"`

	ppid  int
	store beacon.Store
	out   io.Writer
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
	if c.out == nil {
		c.out = os.Stdout
	}
	return beacon.New(c.store, c.out), nil
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
