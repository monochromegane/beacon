package beacon

import (
	"fmt"
	"io"

	"github.com/monochromegane/beacon/internal/context"
)

// Beacon provides the core business logic for managing beacon state files.
type Beacon struct {
	store        Store
	contextStore context.ContextStore
	out          io.Writer
}

// New creates a new Beacon with the given store and output writer.
func New(store Store, out io.Writer) *Beacon {
	return &Beacon{
		store: store,
		out:   out,
	}
}

// NewWithContextStore creates a new Beacon with both stores and output writer.
func NewWithContextStore(store Store, contextStore context.ContextStore, out io.Writer) *Beacon {
	return &Beacon{
		store:        store,
		contextStore: contextStore,
		out:          out,
	}
}

// Emit creates or updates a beacon state file for the given PID.
func (b *Beacon) Emit(pid int, message string) error {
	return b.store.Write(pid, message)
}

// EmitWithContext creates or updates a beacon state file and context file for the given PID.
func (b *Beacon) EmitWithContext(pid int, message string, ctx context.Context) error {
	if err := b.store.Write(pid, message); err != nil {
		return err
	}
	if b.contextStore != nil && ctx != nil {
		return b.contextStore.Write(pid, ctx)
	}
	return nil
}

// Silence removes the beacon state file and context file for the given PID.
func (b *Beacon) Silence(pid int) error {
	if err := b.store.Delete(pid); err != nil {
		return err
	}
	if b.contextStore != nil {
		return b.contextStore.Delete(pid)
	}
	return nil
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
