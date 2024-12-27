package commands

import (
	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/signal"
)

type KillOpts struct {
	ID     string
	Signal string
}

func Kill(opts *KillOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return err
	}

	s := signal.FromString(opts.Signal)

	return cntr.Kill(s)
}
