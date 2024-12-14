package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

const containers = "/var/lib/brownie/containers"

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
}

func Create(opts *CreateOpts, log *zerolog.Logger) error {
	if _, err := os.Stat(filepath.Join(
		containers, opts.ID,
	)); err == nil {
		return fmt.Errorf(
			"container already exists (%s): %w",
			opts.ID, err,
		)
	}

	cntr, err := container.New(
		opts.ID,
		opts.Bundle,
		&container.ContainerOpts{
			PIDFile:       opts.PIDFile,
			ConsoleSocket: opts.ConsoleSocket,
			Stdin:         os.Stdin,
			Stdout:        os.Stdout,
			Stderr:        os.Stderr,
		},
	)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	return cntr.Init("/proc/self/exe", "reexec", log)
}
