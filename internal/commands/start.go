package commands

import (
	"database/sql"
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

const containerSockFilename = "container.sock"

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts, log *zerolog.Logger, db *sql.DB) error {
	cntr, err := container.Load(opts.ID, log, db)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	return cntr.Start()
}
