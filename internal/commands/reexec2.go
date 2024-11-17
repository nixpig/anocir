package commands

import (
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

func Reexec2(opts *ReexecOpts, log *zerolog.Logger, db *database.DB) error {
	bundle, err := db.GetBundleFromID(opts.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get bundle from ID")
		return err
	}

	cntr, err := container.Load(bundle)
	if err != nil {
		log.Error().Err(err).Msg("failed to load bundle")
		return err
	}

	if err := cntr.Reexec2(log); err != nil {
		log.Error().Err(err).Msg("reexec 2 failed...")
		return fmt.Errorf("reexec 2 container: %w", err)
	}

	return nil
}
