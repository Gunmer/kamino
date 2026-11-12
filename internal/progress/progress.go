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

// Cancel reports whether the user requested cancellation.
type Cancel func() bool

// Emit notifies one step when a callback is present.
func Emit(callback Callback, phase Phase, current, total int, path string) {
	if callback != nil {
		callback(Event{Phase: phase, Current: current, Total: total, Path: path})
	}
}

// EnsureNotCancelled aborts cooperatively when the UI requested it.
func EnsureNotCancelled(cancelled Cancel) error {
	if cancelled != nil && cancelled() {
		return &kerrors.OperationCancelled{
			KaminoError: kerrors.KaminoError{Msg: "operación cancelada"},
		}
	}
	return nil
}

// ContextCancel adapts a context to the cancellation callback contract.
func ContextCancel(ctx context.Context) Cancel {
	return func() bool {
		select {
		case <-ctx.Done():
			return true
		default:
			return false
		}
	}
}
