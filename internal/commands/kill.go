package commands

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

func Kill(opts *container.KillOpts, log *zerolog.Logger, db *sql.DB) error {
	log.Info().Str("container id", opts.ID).Msg("killing container...")
	cntr, err := container.Load(opts.ID, log, db)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	return cntr.Kill(opts, log)
}
