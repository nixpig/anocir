package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/nixpig/brownie/internal/container"
	"github.com/rs/zerolog"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts, log *zerolog.Logger) error {
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !opts.Force && !cntr.CanBeDeleted() {
		return errors.New("container cannot be deleted in current state")
	}

	if err := os.Remove(cntr.SockAddr); err != nil {
		return fmt.Errorf("remove ipc socket: %w", err)
	}

	if err := os.RemoveAll(cntr.Path); err != nil {
		return fmt.Errorf("remove container path: %s", err)
	}

	if err := cntr.ExecHooks("poststop"); err != nil {
		log.Warn().Err(err).Msg("execute poststop hooks")
	}

	return nil
}
