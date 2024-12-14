package commands

import (
	"github.com/nixpig/brownie/container"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return err
	}

	return cntr.Delete(opts.Force)
}
