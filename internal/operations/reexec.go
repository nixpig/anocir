package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

type ReexecOpts struct {
	ID              string
	ConsoleSocketFD *int
}

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
