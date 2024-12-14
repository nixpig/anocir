package commands

import (
	"fmt"

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

	s, err := signal.FromString(opts.Signal)
	if err != nil {
		return fmt.Errorf("failed to convert to signal: %w", err)
	}

	return cntr.Kill(s)
}
