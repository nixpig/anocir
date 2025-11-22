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
	return container.WithLock(
		opts.ID,
		opts.RootDir,
		func(c *container.Container) error {
			if err := c.Start(); err != nil {
				return fmt.Errorf("start container: %w", err)
			}

			return nil
		},
	)
}
