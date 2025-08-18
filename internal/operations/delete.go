package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// DeleteOpts holds the options for the Delete operation.
type DeleteOpts struct {
	ID    string
	Force bool
}

// Delete removes a container. It takes DeleteOpts as input, which includes the
// container ID and a force flag.
func Delete(opts *DeleteOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Delete(opts.Force); err != nil {
		return fmt.Errorf("delete container: %w", err)
	}

	return nil
}
