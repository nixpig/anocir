package commands

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
	_, err := db.Query(`select id_ from containers_ where id_ = $id`, sql.Named("id", opts.ID))
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

	b, err := os.ReadFile(filepath.Join(cntr.State.Bundle, "config.json"))
	if err != nil {
		return fmt.Errorf("read bundle config: %w", err)
	}

	query := `insert into containers_ (
		id_, version_, bundle_, pid_, status_, config_
	) values (
		$id, $version, $bundle, $pid, $status, $config
	)`
	if _, err := db.Exec(
		query,
		sql.Named("id", cntr.State.ID),
		sql.Named("version", cntr.State.Version),
		sql.Named("bundle", cntr.State.Bundle),
		sql.Named("pid", cntr.State.PID),
		sql.Named("status", cntr.State.Status),
		sql.Named("config", string(b)),
	); err != nil {
		return fmt.Errorf("insert into db: %w", err)
	}

	return cntr.Init("/proc/self/exe", "fork")
}
