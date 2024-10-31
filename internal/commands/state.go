package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

type StateOpts struct {
	ID string
}

func State(opts *StateOpts, log *zerolog.Logger, db *sql.DB) (string, error) {
	log.Info().Msg("get state...")
	row := db.QueryRow(
		`select bundle_ from containers_ where id_ = $id`,
		sql.Named("id", opts.ID),
	)

	var bundle string
	if err := row.Scan(
		&bundle,
	); err != nil {
		return "", fmt.Errorf("scan container to struct: %w", err)
	}

	log.Info().Msg("loading container")
	cntr, err := container.Load(bundle)
	if err != nil {
		return "", err
	}

	if err := cntr.RefreshState(); err != nil {
		return "", err
	}

	s, err := json.Marshal(cntr.State)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal state")
		return "", fmt.Errorf("marshal state: %w", err)
	}

	return string(s), nil
}
