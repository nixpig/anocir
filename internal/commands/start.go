package commands

import (
	"database/sql"
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts, log *zerolog.Logger, db *sql.DB) error {
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

	return cntr.Start()
}
