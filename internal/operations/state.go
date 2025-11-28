package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

// StateOpts holds the options for the State operation.
type StateOpts struct {
	// ID is the Container ID.
	ID string
	// RootDir is the root directory of the Container state file.
	RootDir string
}

// State returns the state of a Container.
func State(opts *StateOpts) (string, error) {
	var state string
	var err error

	c, err := container.Load(opts.ID, opts.RootDir)
	if err != nil {
		return "", fmt.Errorf("load container: %w", err)
	}

	if err := c.DoWithLock(func(c *container.Container) error {
		state, err = c.GetState()
		if err != nil {
			return fmt.Errorf("state: %w", err)
		}
		return nil
	}); err != nil {
		return "", fmt.Errorf("with lock: %w", err)
	}

	return state, nil
}
