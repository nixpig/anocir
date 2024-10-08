package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/internal/container"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
}

func Create(opts *CreateOpts, log *zerolog.Logger) error {
	root := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)

	cntr, err := container.New(
		opts.ID,
		opts.Bundle,
		root,
		specs.StateCreating,
	)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	if err := cntr.ExecHooks("createRuntime"); err != nil {
		log.Error().Err(err).Msg("failed to execute createRuntime hooks")
		return fmt.Errorf("execute createruntime hooks: %w", err)
	}

	if err := cntr.ExecHooks("createContainer"); err != nil {
		log.Error().Err(err).Msg("failed to execute createContainer hooks")
		return fmt.Errorf("execute createcontainer hooks: %w", err)
	}

	// FIXME: ??? this isn't the correct PID - it should be from _inside_ container, 0 ??
	if err := cntr.Init(opts.ConsoleSocket); err != nil {
		return fmt.Errorf("init container: %w", err)
	}

	pid, err := cntr.Fork(
		opts.PIDFile,
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)
	if err != nil {
		return fmt.Errorf("fork container: %w", err)
	}

	fmt.Println("pid: ", pid)

	return nil
}
