package operations

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nixpig/anocir/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
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
	var state []byte

	err := container.WithLock(
		opts.ID,
		opts.RootDir,
		func(c *container.Container) error {
			// TODO: Probably want to move this into a Container.State() function.
			process, err := os.FindProcess(c.State.Pid)
			if err != nil {
				return fmt.Errorf("find container process: %w", err)
			}

			if err := process.Signal(unix.Signal(0)); err != nil {
				c.State.Status = specs.StateStopped
				if err := c.Save(); err != nil {
					return fmt.Errorf("save stopped state: %w", err)
				}
			}

			state, err = json.Marshal(c.State)
			if err != nil {
				return fmt.Errorf("marshal state: %w", err)
			}

			return nil
		},
	)

	return string(state), err
}
