package commands

import (
	"errors"
	"fmt"

	"github.com/nixpig/brownie/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type KillOpts struct {
	ID     string
	Signal string
}

func Kill(opts *KillOpts, log *zerolog.Logger) error {
	root := container.GetRoot(opts.ID)

	cntr, err := container.Load(root)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	if !cntr.CanBeKilled() {
		log.Error().Msg("container cannot be killed")
		return errors.New("container cannot be killed in current state")
	}

	log.Info().Str("signal", opts.Signal).Msg("get kill signal")
	// s, err := signal.FromString(opts.Signal)
	// if err != nil {
	// 	log.Error().Str("signal", opts.Signal).Msg("failed to convert to signal")
	// 	return fmt.Errorf("failed to convert to signal: %w", err)
	// }

	// FIXME: this is wrong - it needs to send the signal to the process _in_ the container, not the container process itself
	// if err := syscall.Kill(cntr.State.Pid, s); err != nil {
	// 	log.Error().Err(err).Msg("failed to execute kill syscall")
	// 	return fmt.Errorf("failed to execute kill syscall: %w", err)
	// }

	cntr.State.Status = specs.StateStopped
	if err := cntr.State.Save(root); err != nil {
		log.Error().Err(err).Msg("failed to save stopped state")
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	return nil
}
