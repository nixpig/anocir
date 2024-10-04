package commands

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/nixpig/brownie/internal/container"
	"github.com/nixpig/brownie/internal/signal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type KillOpts struct {
	ID     string
	Signal string
}

func Kill(opts *KillOpts, log *zerolog.Logger) error {
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !cntr.CanBeKilled() {
		return errors.New("container cannot be killed in current state")
	}

	s, err := signal.FromString(opts.Signal)
	if err != nil {
		return fmt.Errorf("convert to signal: %w", err)
	}

	if err := syscall.Kill(cntr.State.Pid, s); err != nil {
		return fmt.Errorf("kill container process: %w", err)
	}

	cntr.State.Set(specs.StateStopped)
	if err := cntr.State.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
