package commands

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
}

func Create(opts *CreateOpts, log *zerolog.Logger, db *sql.DB) error {
	_, err := db.Query(
		`select id_ from containers_ where id_ = $id`,
		sql.Named("id", opts.ID),
	)
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

	query := `insert into containers_ (
		id_, bundle_
	) values (
		$id, $bundle
	)`
	if _, err := db.Exec(
		query,
		sql.Named("id", cntr.State.ID),
		sql.Named("bundle", cntr.State.Bundle),
	); err != nil {
		return fmt.Errorf("insert into db: %w", err)
	}

	return cntr.Init("/proc/self/exe", "fork")
}
