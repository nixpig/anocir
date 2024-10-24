package commands

import (
	"database/sql"
	"errors"
	"fmt"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/internal/container"
	"github.com/nixpig/brownie/internal/container/signal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type KillOpts struct {
	ID     string
	Signal string
}

func Kill(opts *KillOpts, log *zerolog.Logger, db *sql.DB) error {
	log.Info().Str("container id", opts.ID).Msg("killing container...")
	cntr, err := container.Load(opts.ID, log, db)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	log.Info().Str("status", string(cntr.State.Status)).Msg("🏗️ CURRENT STATUS...")
	if !cntr.CanBeKilled() {
		log.Error().Str("state", string(cntr.State.Status)).Msg("container cannot be killed")
		return errors.New("container cannot be killed in current state")
	}

	s, err := signal.FromString(opts.Signal)
	if err != nil {
		log.Error().Str("signal", opts.Signal).Msg("failed to convert to signal")
		return fmt.Errorf("failed to convert to signal: %w", err)
	}

	if err := syscall.Kill(cntr.State.PID, s); err != nil {
		log.Error().Err(err).Int("pid", cntr.State.PID).Any("signal", s).Msg("failed to execute kill syscall")
		return fmt.Errorf("failed to execute kill syscall: %w", err)
	}

	cntr.State.Status = specs.StateStopped
	if err := cntr.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save stopped state")
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	return nil
}
