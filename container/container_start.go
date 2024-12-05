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

	conn, err := net.Dial("unix", filepath.Join("/var/lib/brownie/containers", c.ID(), containerSockFilename))
	if err != nil {
		return fmt.Errorf("dial socket: %w", err)
	}

	if err := c.ExecHooks("prestart", log); err != nil {
		// TODO: run DELETE tasks here, then...
		if err := c.ExecHooks("poststop", log); err != nil {
			log.Warn().Err(err).Msg("failed to execute poststop hooks")
			fmt.Println("WARNING: failed to execute poststop hooks")
		}

		return fmt.Errorf("failed to run prestart hooks: %w", err)
	}

	log.Info().Msg("sending start")
	if _, err := conn.Write([]byte("start")); err != nil {
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()

	return nil
}
