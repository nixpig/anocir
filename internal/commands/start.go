package commands

import (
	"errors"
	"fmt"
	"net"

	"github.com/nixpig/brownie/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts, log *zerolog.Logger) error {
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !cntr.CanBeStarted() {
		return errors.New("container cannot be started in current state")
	}

	if err := cntr.ExecHooks("startContainer"); err != nil {
		return fmt.Errorf("execute startcontainer hooks: %w", err)
	}

	conn, err := net.Dial("unix", cntr.SockAddr)
	if err != nil {
		log.Error().Err(err).Msg("start: dial socket")
		return fmt.Errorf("dial socket: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("start: send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}

	cntr.State.Set(specs.StateRunning)
	if err := cntr.State.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save state")
	}

	if err := cntr.ExecHooks("poststart"); err != nil {
		log.Warn().Err(err).Msg("execute poststart hooks")
	}

	cntr.State.Set(specs.StateStopped)
	if err := cntr.State.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
