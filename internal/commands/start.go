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
	log.Info().Any("opts", opts).Msg("run start command")
	log.Info().Str("id", opts.ID).Msg("load container")
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	log.Info().Msg("check if container can be started")
	if !cntr.CanBeStarted() {
		log.Error().Msg("container cannot be started in current state")
		return errors.New("container cannot be started in current state")
	}

	log.Info().Msg("execute startContainer hooks")
	if err := cntr.ExecHooks("startContainer"); err != nil {
		log.Error().Err(err).Msg("failed to execute startContainer hooks")
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	log.Info().Str("sockaddr", cntr.SockAddr).Msg("dial container sockaddr")
	conn, err := net.Dial("unix", cntr.SockAddr)
	if err != nil {
		log.Error().Err(err).Msg("failed to dial container sockaddr")
		return fmt.Errorf("dial socket: %w", err)
	}

	log.Info().Msg("send start message")
	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("failed to send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()

	log.Info().
		Any("state", cntr.State.Status).
		Msg("set and save running state")
	cntr.State.Set(specs.StateRunning)
	if err := cntr.State.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save state")
	}

	log.Info().Msg("execute poststart hooks")
	if err := cntr.ExecHooks("poststart"); err != nil {
		log.Warn().Err(err).Msg("failed to execute poststart hooks")
	}

	log.Info().
		Any("state", cntr.State.Status).
		Msg("set and save stopped state")
	cntr.State.Set(specs.StateStopped)
	if err := cntr.State.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
