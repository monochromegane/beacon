package main

import (
	"fmt"
	"os"

	"github.com/monochromegane/beacon/cmd"
)

func main() {
	ppid := os.Getppid()
	cli := cmd.NewCLI(ppid)
	if err := cli.Execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
