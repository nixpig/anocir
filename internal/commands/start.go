package commands

import (
	"fmt"
	"time"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
)

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts, log *zerolog.Logger, db *database.DB) error {
	bundle, err := db.GetBundleFromID(opts.ID)
	if err != nil {
		return err
	}

	cntr, err := container.Load(bundle)
	if err != nil {
		return err
	}

	if err := cntr.Start(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	time.Sleep(time.Second * 3)
	fmt.Println(cntr.Status())
	return nil
}
