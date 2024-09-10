package cmd

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/nixpig/brownie/pkg"
)

func Kill(containerID, signal string) error {
	state, err := pkg.GetState(containerID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
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
