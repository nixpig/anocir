package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// KillOpts holds the options for the Kill operation.
type KillOpts struct {
	ID     string
	Signal string
}

// Kill sends a signal to a container. It takes KillOpts as input, which
// includes the container ID and the signal to send.
func Kill(opts *KillOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Kill(opts.Signal); err != nil {
		return fmt.Errorf("kill container: %w", err)
	}

	return nil
}
