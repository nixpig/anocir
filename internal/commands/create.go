package commands

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/nixpig/brownie/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
}

func Create(opts *CreateOpts, log *zerolog.Logger, db *sql.DB) error {
	cntr, err := container.New(
		opts.ID,
		opts.Bundle,
		specs.StateCreating,
		log,
		db,
	)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	if err := cntr.ExecHooks("createRuntime"); err != nil {
		log.Error().Err(err).Msg("failed to execute createRuntime hooks")
		return fmt.Errorf("execute createruntime hooks: %w", err)
	}

	if err := cntr.ExecHooks("createContainer"); err != nil {
		log.Error().Err(err).Msg("failed to execute createContainer hooks")
		return fmt.Errorf("execute createcontainer hooks: %w", err)
	}

	// FIXME: ??? this isn't the correct PID - it should be from _inside_ container, 0 ??
	pid, err := cntr.Init(&container.InitOpts{
		PIDFile:       opts.PIDFile,
		ConsoleSocket: opts.ConsoleSocket,
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	}, log)
	if err != nil {
		log.Error().Err(err).Msg("failed to init container")
		return fmt.Errorf("init container: %w", err)
	}

	log.Info().Int("pid", pid).Msg("initialised container")

	return nil
}
