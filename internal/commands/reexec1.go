package commands

import (
	"fmt"

	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

type ReexecOpts struct {
	ID string
}

func Reexec1(opts *ReexecOpts, log *zerolog.Logger) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return err
	}

	if err := cntr.Reexec1(log); err != nil {
		log.Error().Err(err).Msg("reexec 1 failed...")
		return fmt.Errorf("reexec 1 container: %w", err)
	}

	return nil
}
