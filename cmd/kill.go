package cmd

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Kill(containerID, signal string) error {
	state, err := pkg.GetState(containerID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	if state.Status != specs.StateCreated && state.Status != specs.StateRunning {
		return errors.New("container is not created or running")
	}

	s, err := internal.ToSignal(signal)
	if err != nil {
		return err
	}

	// FIXME: send signal provided
	if err := syscall.Kill(state.Pid, s); err != nil {
		return fmt.Errorf("kill container process: %w", err)
	}

	state.Status = specs.StateStopped
	saveState(state)

	return nil
}
