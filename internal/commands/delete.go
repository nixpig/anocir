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
	log.Info().Any("opts", opts).Msg("run delete command")
	log.Info().Str("id", opts.ID).Msg("load container")
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	log.Info().Msg("check if container can be deleted")
	if !opts.Force && !cntr.CanBeDeleted() {
		log.Error().Msg("container cannot be deleted in current state")
		return errors.New("container cannot be deleted in current state")
	}

	log.Info().Int("pid", cntr.State.Pid).Msg("kill the container host process")
	if err := syscall.Kill(cntr.State.Pid, 9); err != nil {
		log.Error().
			Err(err).
			Int("pid", cntr.State.Pid).
			Msg("failed to kill the container host process")
		return fmt.Errorf("kill container host process")
	}

	log.Info().Str("sockaddr", cntr.SockAddr).Msg("remove container sockaddr")
	if err := os.Remove(cntr.SockAddr); err != nil {
		log.Error().
			Err(err).
			Str("sockaddr", cntr.SockAddr).
			Msg("failed to remove container sockaddr")
		return fmt.Errorf("remove ipc socket: %w", err)
	}

	log.Info().Str("path", cntr.Path).Msg("remove container path")
	if err := os.RemoveAll(cntr.Path); err != nil {
		log.Error().
			Err(err).
			Str("path", cntr.Path).
			Msg("failed to remove container path")
		return fmt.Errorf("remove container path: %s", err)
	}

	log.Info().Msg("execute poststop hooks")
	if err := cntr.ExecHooks("poststop"); err != nil {
		log.Warn().Err(err).Msg("failed to execute poststop hooks")
	}

	return nil
}
