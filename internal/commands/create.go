package commands

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/nixpig/brownie/container"
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

	return cntr.Init(&container.InitOpts{
		PIDFile:       opts.PIDFile,
		ConsoleSocket: opts.ConsoleSocket,
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	}, log)
}
