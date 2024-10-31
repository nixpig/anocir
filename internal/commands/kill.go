package commands

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/container/signal"
	"github.com/rs/zerolog"
)

type KillOpts struct {
	ID     string
	Signal string
}

func Kill(opts *KillOpts, log *zerolog.Logger, db *sql.DB) error {
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
	log.Info().Str("container id", opts.ID).Msg("killing container...")

	s, err := signal.FromString(opts.Signal)
	if err != nil {
		return fmt.Errorf("failed to convert to signal: %w", err)
	}

	return cntr.Kill(s)
}
