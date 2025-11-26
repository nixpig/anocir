package container

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// GetState returns the state of the container. In the case the container
// process no longer exists, it has the side effect of internally modifying
// the state to be 'stopped' before returning.
func (c *Container) GetState() (string, error) {
	if c.State.Pid != 0 {
		process, err := os.FindProcess(c.State.Pid)
		if err != nil {
			return "", fmt.Errorf("find container process: %w", err)
		}

		if err := process.Signal(unix.Signal(0)); err != nil {
			c.State.Status = specs.StateStopped
			if err := c.Save(); err != nil {
				return "", fmt.Errorf("save stopped state: %w", err)
			}
		}
	}

	state, err := json.Marshal(c.State)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}

	return string(state), nil
}
