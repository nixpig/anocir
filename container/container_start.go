package container

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Start(log *zerolog.Logger) error {
	if c.Spec.Process == nil {
		c.SetStatus(specs.StateStopped)
		if err := c.Save(); err != nil {
			return fmt.Errorf("(start 1) write state file: %w", err)
		}
		return nil
	}

	if !c.CanBeStarted() {
		return errors.New("container cannot be started in current state")
	}

	conn, err := net.Dial("unix", filepath.Join(containerRootDir, c.ID(), containerSockFilename))
	if err != nil {
		return fmt.Errorf("dial socket: %w", err)
	}

	// FIXME: 'prestart' hook is deprecated and appears to break 'docker run'??
	if err := c.ExecHooks("prestart"); err != nil {
		// TODO: run DELETE tasks here, then...
		if err := c.ExecHooks("poststop"); err != nil {
			log.Warn().Err(err).Msg("failed to execute poststop hooks")
			fmt.Println("WARNING: failed to execute poststop hooks")
		}

		log.Warn().Err(err).Msg("failed to execute prestart hooks")
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()
	c.SetStatus(specs.StateRunning)
	if err := c.Save(); err != nil {
		// do something with err??
		log.Error().Err(err).Msg("⁉️ host save state running")
		fmt.Println(err)
		return fmt.Errorf("save host container state: %w", err)
	}
	// FIXME: do these need to move up before the cmd.Wait call??
	if err := c.ExecHooks("poststart"); err != nil {
		// TODO: how to handle this (log a warning) from start command??
		// FIXME: needs to 'log a warning'
		log.Warn().Err(err).Msg("failed to execute poststart hook")
		fmt.Println("WARNING: ", err)
	}

	return nil
}
