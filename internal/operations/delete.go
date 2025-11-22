package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// DeleteOpts holds the options for the Delete operation.
type DeleteOpts struct {
	// ID is the Container ID.
	ID string
	// RootDir is the root directory containing the Container state file.
	RootDir string
	// Force is used to override the conditions for deleting a Container.
	Force bool
}

// Delete removes a container.
func Delete(opts *DeleteOpts) error {
	return container.WithLock(
		opts.ID,
		opts.RootDir,
		func(c *container.Container) error {
			if err := c.Delete(opts.Force); err != nil {
				return fmt.Errorf("delete container: %w", err)
			}

			return nil
		},
	)
}
