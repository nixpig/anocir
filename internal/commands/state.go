package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type StateOpts struct {
	ID string
}

func State(opts *StateOpts, log *zerolog.Logger, db *database.DB) (string, error) {
	bundle, err := db.GetBundleFromID(opts.ID)
	if err != nil {
		return "", err
	}

	cntr, err := container.Load(bundle)
	if err != nil {
		return "", err
	}

	if err := cntr.RefreshState(log); err != nil {
		return "", err
	}

	s, err := json.Marshal(cntr.State)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal state")
		return "", fmt.Errorf("marshal state: %w", err)
	}

	return string(s), nil
}
