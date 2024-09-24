package commands

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Kill(containerID, signal string) error {
	container, err := internal.LoadContainer(containerID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if container.State.Status != specs.StateCreated && container.State.Status != specs.StateRunning {
		return errors.New("container is not created or running")
	}

	s, err := internal.ToSignal(signal)
	if err != nil {
		return fmt.Errorf("convert to signal: %w", err)
	}

	container.State.Set(specs.StateStopped)
	if err := container.State.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	if err := syscall.Kill(container.State.Pid, s); err != nil {
		return fmt.Errorf("kill container process: %w", err)
	}

	return nil
}
