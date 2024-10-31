package commands

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts, log *zerolog.Logger, db *sql.DB) error {
	row := db.QueryRow(
		`select bundle_ from containers_ where id_ = $id`,
		sql.Named("id", opts.ID),
	)

	var bundle string
	if err := row.Scan(
		&bundle,
	); err != nil {
		return fmt.Errorf("scan container to struct: %w", err)
	}

	log.Info().Msg("loading container")
	cntr, err := container.Load(bundle)
	if err != nil {
		return err
	}

	if err := cntr.Delete(); err != nil {
		return err
	}

	res, err := db.Exec(
		`delete from containers_ where id_ = $id`,
		sql.Named("id", opts.ID),
	)
	if err != nil {
		return fmt.Errorf("delete container db: %w", err)
	}

	if c, err := res.RowsAffected(); err != nil || c == 0 {
		return errors.New("didn't delete container for whatever reason")
	}

	return nil
}
