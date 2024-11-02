package commands

import (
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type ReexecOpts struct {
	ID string
}

func Reexec(opts *ReexecOpts, log *zerolog.Logger, db *database.DB) error {
	bundle, err := db.GetBundleFromID(opts.ID)
	if err != nil {
		return err
	}

	cntr, err := container.Load(bundle)
	if err != nil {
		return err
	}

	if err := cntr.Reexec(); err != nil {
		return fmt.Errorf("reexec container: %w", err)
	}

	return nil
}
