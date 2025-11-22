package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// ReexecOpts holds the options for the Reexec operation.
type ReexecOpts struct {
	// ID is the Container ID.
	ID string
	// RootDir is the root directory for the Container state file.
	RootDir string
	// ConsoleSocketFD is the file descriptor of the unix domain socket used to
	// recieve the PTY master file descriptor sent by the container runtime.
	ConsoleSocketFD *int
}

// Reexec re-executes the container process.
func Reexec(opts *ReexecOpts) error {
	cntr, err := container.Load(opts.ID, opts.RootDir)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	cntr.ConsoleSocketFD = opts.ConsoleSocketFD

	if err := cntr.Reexec(); err != nil {
		return fmt.Errorf("reexec container: %w", err)
	}

	return nil
}
