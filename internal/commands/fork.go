package commands

import (
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type ForkOpts struct {
	ID string
}

func Fork(opts *ForkOpts, log *zerolog.Logger, db *database.DB) error {
	bundle, err := db.GetBundleFromID(opts.ID)
	if err != nil {
		return err
	}

	cntr, err := container.Load(bundle)
	if err != nil {
		return err
	}

	if err := cntr.Fork(); err != nil {
		return fmt.Errorf("fork container: %w", err)
	}

	return nil
}
