package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// StartOpts holds the options for the Start operation.
type StartOpts struct {
	ID string
}

// Start starts a container. It takes StartOpts as input, which include the
// container ID.
func Start(opts *StartOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Start(); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	return nil
}
