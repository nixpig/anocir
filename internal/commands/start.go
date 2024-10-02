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
	container, err := container.LoadContainer(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !container.CanBeStarted() {
		return errors.New("container cannot be started in current state")
	}

	if err := container.ExecHooks("startContainer"); err != nil {
		return fmt.Errorf("execute startcontainer hooks: %w", err)
	}

	conn, err := net.Dial("unix", container.SockAddr)
	if err != nil {
		log.Error().Err(err).Msg("start: dial socket")
		return fmt.Errorf("dial socket: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("start: send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}

	container.State.Set(specs.StateRunning)
	if err := container.State.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save state")
	}

	if err := container.ExecHooks("poststart"); err != nil {
		return fmt.Errorf("execute poststart hooks: %w", err)
	}

	container.State.Set(specs.StateStopped)
	if err := container.State.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
