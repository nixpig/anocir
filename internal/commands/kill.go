package commands

import (
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/container/signal"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type KillOpts struct {
	ID     string
	Signal string
}

func Kill(opts *KillOpts, log *zerolog.Logger, db *database.DB) error {

	bundle, err := db.GetBundleFromID(opts.ID)
	if err != nil {
		return err
	}

	cntr, err := container.Load(bundle)
	if err != nil {
		return err
	}

	s, err := signal.FromString(opts.Signal)
	if err != nil {
		return fmt.Errorf("failed to convert to signal: %w", err)
	}

	return cntr.Kill(s)
}
