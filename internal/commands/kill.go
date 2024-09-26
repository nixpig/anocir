package commands

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type KillOpts struct {
	ID     string
	Signal string
}

func Kill(opts *KillOpts, log *zerolog.Logger) error {
	container, err := internal.LoadContainer(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !container.CanBeKilled() {
		return errors.New("container is not created or running")
	}

	s, err := internal.ToSignal(opts.Signal)
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
