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
	cntr, err := container.Load(opts.ID, opts.RootDir)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Lock(); err != nil {
		return fmt.Errorf("lock container: %w", err)
	}
	defer cntr.Unlock()

	if err := cntr.Delete(opts.Force); err != nil {
		return fmt.Errorf("delete container: %w", err)
	}

	return nil
}
