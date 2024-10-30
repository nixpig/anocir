package container

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type StartOpts struct {
	ID string
}

func (c *Container) Start(opts *StartOpts, log *zerolog.Logger) error {
	if c.Spec.Process == nil {
		log.Info().Msg("NO PROCESS SET!!")
		c.State.Status = specs.StateStopped
		if err := c.Save(); err != nil {
			log.Error().Err(err).Msg("failed to write state file")
			return fmt.Errorf("write state file: %w", err)
		}
		return nil
	}

	if !c.CanBeStarted() {
		log.Error().Msg("container cannot be started in current state")
		return errors.New("container cannot be started in current state")
	}

	if err := c.ExecHooks("startContainer"); err != nil {
		log.Error().Err(err).Msg("failed to execute startContainer hooks")
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	conn, err := net.Dial("unix", filepath.Join(c.State.Bundle, containerSockFilename))
	if err != nil {
		log.Error().Err(err).Msg("failed to dial container sockaddr")
		return fmt.Errorf("dial socket: %w", err)
	}

	if err := c.ExecHooks("prestart"); err != nil {
		log.Error().Err(err).Msg("failed to execute prestart hooks")
		c.State.Status = specs.StateStopped
		if err := c.Save(); err != nil {
			log.Error().Err(err).Msg("failed to write state file")
			return fmt.Errorf("write state file: %w", err)
		}
		log.Info().Msg("BEFORE FAIL DELETE")

		// TODO: run DELETE tasks here, then...

		if err := c.ExecHooks("poststop"); err != nil {
			fmt.Println("WARNING: failed to execute poststop hooks")
			log.Warn().Err(err).Msg("failed to execute poststop hooks")
		}

		return errors.New("failed to run prestart hooks")
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("failed to send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()

	if err := c.ExecHooks("poststart"); err != nil {
		log.Warn().Err(err).Msg("failed to execute poststart hooks")
		// TODO: how to handle this (log a warning) from start command??
	}

	// FIXME: ?? when process starts, the PID in state should be updated to the process IN the container??

	// c.State.Status = specs.StateStopped
	// if err := c.Save(); err != nil {
	// 	log.Error().Err(err).Msg("save state after stopped")
	// 	return fmt.Errorf("failed to save stopped state: %w", err)
	// }

	return nil
}
