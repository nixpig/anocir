package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// ReexecOpts holds the options for the Reexec operation.
type ReexecOpts struct {
	ID              string
	ConsoleSocketFD *int
}

// Reexec re-executes the container process. It takes ReexecOpts as input,
// which includes the container ID and a console socket file descriptor.
func Reexec(opts *ReexecOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	cntr.ConsoleSocketFD = opts.ConsoleSocketFD

	if err := cntr.Reexec(); err != nil {
		return fmt.Errorf("reexec container: %w", err)
	}

	return nil
}
