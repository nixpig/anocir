package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// StartOpts holds the options for the Start operation.
type StartOpts struct {
	// ID is the Container ID.
	ID string
	// RootDir is the root directory of the Container state file.
	RootDir string
}

// Start starts a Container.
func Start(opts *StartOpts) error {
	cntr, err := container.Load(opts.ID, opts.RootDir)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Start(); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	return nil
}
