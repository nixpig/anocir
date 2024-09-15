package commands

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Kill(containerID, signal string) error {
	fmt.Println("get state")
	state, err := internal.GetState(containerID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	fmt.Println("check status")
	if state.Status != specs.StateCreated && state.Status != specs.StateRunning {
		return errors.New("container is not created or running")
	}

	fmt.Println("convert to signal")
	s, err := internal.ToSignal(signal)
	if err != nil {
		return fmt.Errorf("convert to signal: %w", err)
	}

	fmt.Println("save state")
	state.Status = specs.StateStopped
	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	if err := syscall.Kill(state.Pid, s); err != nil {
		return fmt.Errorf("kill container process: %w", err)
	}

	fmt.Println("return from kill")
	return nil
}
