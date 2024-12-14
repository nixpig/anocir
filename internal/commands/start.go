package commands

import (
	"github.com/nixpig/brownie/container"
)

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return err
	}

	return cntr.Start()
}
