package commands

import (
	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts, log *zerolog.Logger, db *database.DB) error {
	cntr, err := container.Load(opts.ID)
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
