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
	ID      string
	RootDir string
}

// State returns the state of a container. It takes StateOpts as input,
// which include the container ID.
func State(opts *StateOpts) (string, error) {
	cntr, err := container.Load(opts.ID, opts.RootDir)
	if err != nil {
		return "", fmt.Errorf("load container: %w", err)
	}

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
