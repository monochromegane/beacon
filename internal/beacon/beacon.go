package beacon

import (
	"fmt"
	"io"
)

// Beacon provides the core business logic for managing beacon state files.
type Beacon struct {
	store Store
	out   io.Writer
}

// New creates a new Beacon with the given store and output writer.
func New(store Store, out io.Writer) *Beacon {
	return &Beacon{
		store: store,
		out:   out,
	}
}

// Emit creates or updates a beacon state file for the given PID.
func (b *Beacon) Emit(pid int, message string) error {
	return b.store.Write(pid, message)
}

// Silence removes the beacon state file for the given PID.
func (b *Beacon) Silence(pid int) error {
	return b.store.Delete(pid)
}

// List displays all active beacon states to the output writer.
func (b *Beacon) List() error {
	states, err := b.store.List()
	if err != nil {
		return err
	}
	for _, state := range states {
		fmt.Fprintf(b.out, "%d\t%s\n", state.PID, state.Message)
	}
	return nil
}
