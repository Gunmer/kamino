package install

import (
	"context"
	"os/exec"
)

// Runner isolates the external processes an installation needs (brew, curl,
// sh) and the PATH lookup that decides whether a method is viable. Tests inject
// a fake so no real installer ever runs.
type Runner interface {
	// Lookup reports whether an executable with this name is on the PATH.
	Lookup(name string) bool
	// Run executes name with args and returns its combined output.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// ExecRunner runs real processes.
type ExecRunner struct{}

// Lookup reports whether name resolves on the PATH.
func (ExecRunner) Lookup(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Run executes the command and returns its combined output.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	return command.CombinedOutput()
}
