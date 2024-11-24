package commands

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
}

func Create(opts *CreateOpts, log *zerolog.Logger, db *database.DB) error {
	_, err := db.GetBundleFromID(opts.ID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf(
			"container already exists (%s): %w",
			opts.ID, err,
		)
	}

	cntr, err := container.New(
		opts.ID,
		opts.Bundle,
		&container.ContainerOpts{
			PIDFile:       opts.PIDFile,
			ConsoleSocket: opts.ConsoleSocket,
			Stdin:         os.Stdin,
			Stdout:        os.Stdout,
			Stderr:        os.Stderr,
		},
	)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	if err := db.CreateContainer(opts.ID, opts.Bundle); err != nil {
		return err
	}

	return cntr.Init("/proc/self/exe", "reexec", log)
}
