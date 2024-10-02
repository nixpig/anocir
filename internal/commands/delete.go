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
	container, err := container.LoadContainer(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !opts.Force && !container.CanBeDeleted() {
		return errors.New("container cannot be deleted in current state")
	}

	if err := os.Remove(container.SockAddr); err != nil {
		return fmt.Errorf("remove ipc socket: %w", err)
	}

	if err := os.RemoveAll(container.Path); err != nil {
		return fmt.Errorf("remove container path: %s", err)
	}

	if err := container.ExecHooks("poststop"); err != nil {
		return fmt.Errorf("execute poststop hooks: %w", err)
	}

	return nil
}
