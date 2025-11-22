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
	cntr, err := container.Load(opts.ID, opts.RootDir)
	if err != nil {
		return "", fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Lock(); err != nil {
		return "", fmt.Errorf("lock container: %w", err)
	}
	defer cntr.Unlock()

	process, err := os.FindProcess(cntr.State.Pid)
	if err != nil {
		return "", fmt.Errorf("find container process: %w", err)
	}

	if err := process.Signal(unix.Signal(0)); err != nil {
		cntr.State.Status = specs.StateStopped
		if err := cntr.Save(); err != nil {
			return "", fmt.Errorf("save stopped state: %w", err)
		}
	}

	state, err := json.Marshal(cntr.State)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}

	return string(state), nil
}
