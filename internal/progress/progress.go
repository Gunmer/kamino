// Package progress defines the progress and cancellation contract.
package progress

import (
	"context"

	kerrors "github.com/Gunmer/kamino/internal/errors"
)

// Phase is a visible phase of a long operation.
type Phase string

const (
	Planning    Phase = "planning"
	Verifying   Phase = "verifying"
	Extracting  Phase = "extracting"
	Classifying Phase = "classifying"
	Writing     Phase = "writing"
	Downloading Phase = "downloading"
	Replacing   Phase = "replacing"
)

// Event is one progress step.
type Event struct {
	Phase   Phase
	Current int
	Total   int
	Path    string
}

// Callback receives progress events.
type Callback func(Event)

// Emit notifies one step when a callback is present.
func Emit(callback Callback, phase Phase, current, total int, path string) {
	if callback != nil {
		callback(Event{Phase: phase, Current: current, Total: total, Path: path})
	}
}

// EnsureNotCancelled aborts cooperatively when the context is done.
func EnsureNotCancelled(ctx context.Context) error {
	if ctx.Err() != nil {
		return &kerrors.OperationCancelled{
			KaminoError: kerrors.KaminoError{Msg: "operación cancelada"},
		}
	}
	return nil
}
