package commands

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/nixpig/brownie/internal/container"
	"github.com/rs/zerolog"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts, log *zerolog.Logger) error {
	root := container.GetRoot(opts.ID)

	cntr, err := container.Load(root)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !opts.Force && !cntr.CanBeDeleted() {
		return errors.New("container cannot be deleted in current state")
	}

	process, _ := os.FindProcess(cntr.State.PID)
	if process != nil {
		process.Signal(syscall.Signal(0))
	}

	if err := os.RemoveAll(cntr.Root); err != nil {
		return fmt.Errorf("remove container path: %s", err)
	}

	if err := cntr.ExecHooks("poststop"); err != nil {
		log.Warn().Err(err).Msg("failed to execute poststop hooks")
	}

	return nil
}
