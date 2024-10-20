package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/internal/container"
	"github.com/rs/zerolog"
)

type StateOpts struct {
	ID string
}

func State(opts *StateOpts, log *zerolog.Logger, db *sql.DB) (string, error) {
	cntr, err := container.Load(opts.ID, log, db)
	if err != nil {
		return "", fmt.Errorf("load container in state: %w", err)
	}

	s, err := json.Marshal(cntr.State)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal state")
		return "", fmt.Errorf("marshal state: %w", err)
	}

	return string(s), nil
}
