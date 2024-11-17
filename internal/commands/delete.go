package commands

import (
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts, log *zerolog.Logger, db *database.DB) error {
	bundle, err := db.GetBundleFromID(opts.ID)
	if err != nil {
		return fmt.Errorf("delete container (%s): %w", opts.ID, err)
	}

	cntr, err := container.Load(bundle)
	if err != nil {
		return err
	}

	if err := cntr.Delete(opts.Force); err != nil {
		return err
	}

	if err := db.DeleteContainerByID(opts.ID); err != nil {
		return err
	}

	return nil
}
