package commands

import (
	"github.com/nixpig/brownie/container"
)

type ReexecOpts struct {
	ID string
}

func Reexec(opts *ReexecOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return err
	}

	return cntr.Reexec()
}
