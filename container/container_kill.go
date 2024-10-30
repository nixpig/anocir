package container

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/nixpig/brownie/container/signal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type KillOpts struct {
	ID     string
	Signal string
}

func (c *Container) Kill(opts *KillOpts, log *zerolog.Logger) error {
	log.Info().Str("status", string(c.State.Status)).Msg("🏗️ CURRENT STATUS...")
	if !c.CanBeKilled() {
		log.Error().Str("state", string(c.State.Status)).Msg("container cannot be killed")
		return errors.New("container cannot be killed in current state")
	}

	s, err := signal.FromString(opts.Signal)
	if err != nil {
		log.Error().Str("signal", opts.Signal).Msg("failed to convert to signal")
		return fmt.Errorf("failed to convert to signal: %w", err)
	}

	if err := syscall.Kill(c.State.PID, s); err != nil {
		log.Error().Err(err).Int("pid", c.State.PID).Any("signal", s).Msg("failed to execute kill syscall")
		return fmt.Errorf("failed to execute kill syscall: %w", err)
	}

	c.State.Status = specs.StateStopped
	if err := c.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save stopped state")
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	return nil
}
