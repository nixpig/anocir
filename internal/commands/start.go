package commands

import (
	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts, log *zerolog.Logger, db *database.DB) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return err
	}

	return cntr.Start()
}
