package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/container"
	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	containerRootDir = "/var/lib/brownie/containers"
	configFilename   = "config.json"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
	ReexecCmd     string
	ReexecArgs    []string
}

func Create(opts *CreateOpts) error {
	if container.Exists(containerRootDir, opts.ID) {
		return fmt.Errorf("container with id '%s' already exists", opts.ID)
	}

	bundle, err := filepath.Abs(opts.Bundle)
	if err != nil {
		return fmt.Errorf("absolute path from new container bundle: %w", err)
	}

	b, err := os.ReadFile(filepath.Join(opts.Bundle, configFilename))
	if err != nil {
		return fmt.Errorf("read new container config file: %w", err)
	}

	var spec *specs.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		return fmt.Errorf("parse new container config: %w", err)
	}

	cntr, err := container.New(
		opts.ID,
		spec,
		&container.ContainerOpts{
			Bundle:        bundle,
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

	if err := cntr.Init(opts.ReexecCmd, opts.ReexecArgs); err != nil {
		return fmt.Errorf("initialise container: %w", err)
	}

	if err := cntr.Save(); err != nil {
		return fmt.Errorf("save container: %w", err)
	}

	return nil
}
