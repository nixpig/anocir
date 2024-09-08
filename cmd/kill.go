package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nixpig/brownie/pkg"
)

func Kill(containerID, signal string) error {
	containerPath := filepath.Join(BrownieRootDir, "containers", containerID)

	fc, err := os.ReadFile(filepath.Join(containerPath, "state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("container not found")
		} else {
			return fmt.Errorf("stat container path: %w", err)
		}
	}

	var state pkg.State
	if err := json.Unmarshal(fc, &state); err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if state.Status != pkg.Created && state.Status != pkg.Running {
		return errors.New("container is not created or running")
	}

	// FIXME: send signal provided
	if err := syscall.Kill(*state.PID, syscall.SIGKILL); err != nil {
		return fmt.Errorf("kill container process: %w", err)
	}

	state.Status = pkg.Stopped
	saveState(state)

	return nil
}
