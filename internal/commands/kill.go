package commands

import (
	"errors"
	"fmt"

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
	log.Info().Any("opts", opts).Msg("run kill command")
	log.Info().Str("id", opts.ID).Msg("load container")
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	log.Info().Msg("check if container can be killed")
	if !cntr.CanBeKilled() {
		log.Error().Msg("container cannot be killed")
		return errors.New("container cannot be killed in current state")
	}

	log.Info().Str("signal", opts.Signal).Msg("get kill signal")
	s, err := signal.FromString(opts.Signal)
	if err != nil {
		log.Error().Str("signal", opts.Signal).Msg("failed to convert to signal")
		return fmt.Errorf("failed to convert to signal: %w", err)
	}

	log.Info().
		Int("pid", cntr.State.Pid).
		Str("signal", s.String()).
		Msg("execute kill syscall")
	// FIXME: this is wrong - it needs to send the signal to the process _in_ the container, not the container process itself
	// if err := syscall.Kill(cntr.State.Pid, s); err != nil {
	// 	log.Error().Err(err).Msg("failed to execute kill syscall")
	// 	return fmt.Errorf("failed to execute kill syscall: %w", err)
	// }

	log.Info().
		Any("state", cntr.State.Status).
		Msg("set stopped state and save")
	cntr.State.Set(specs.StateStopped)
	if err := cntr.State.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save stopped state")
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	return nil
}
