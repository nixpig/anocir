package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// KillOpts holds the options for the Kill operation.
type KillOpts struct {
	// ID is the Container ID.
	ID string
	// RootDir is the root directory of the Container state file.
	RootDir string
	// Signal is the signal to send to the Container.
	Signal string
}

// Kill sends a signal to a Container.
func Kill(opts *KillOpts) error {
	cntr, err := container.Load(opts.ID, opts.RootDir)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Lock(); err != nil {
		return fmt.Errorf("lock container: %w", err)
	}

	if err := cntr.Kill(opts.Signal); err != nil {
		return fmt.Errorf("kill container: %w", err)
	}

	return nil
}
