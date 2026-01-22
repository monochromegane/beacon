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

// Emit creates or updates a beacon state file for the given ID.
func (b *Beacon) Emit(id string, message string) error {
	return b.store.Write(id, message)
}

// EmitWithContext creates or updates a beacon state file and context file for the given ID.
func (b *Beacon) EmitWithContext(id string, message string, ctx context.Context) error {
	if err := b.store.Write(id, message); err != nil {
		return err
	}
	if b.contextStore != nil && ctx != nil {
		return b.contextStore.Write(id, ctx)
	}
	return nil
}

// Silence removes the beacon state file and context file for the given ID.
func (b *Beacon) Silence(id string) error {
	if err := b.store.Delete(id); err != nil {
		return err
	}
	if b.contextStore != nil {
		return b.contextStore.Delete(id)
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
		fmt.Fprintf(b.out, "%s\t%s\n", state.ID, state.Message)
	}
	return nil
}
