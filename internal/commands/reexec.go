package commands

import (
	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

type ReexecOpts struct {
	ID string
}

func Reexec(opts *ReexecOpts, log *zerolog.Logger) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return err
	}

	return cntr.Reexec(log)
}
