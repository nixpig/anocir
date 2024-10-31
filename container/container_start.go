package container

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Start() error {
	if c.Spec.Process == nil {
		c.State.Status = specs.StateStopped
		if err := c.hSave(); err != nil {
			return fmt.Errorf("write state file: %w", err)
		}
		return nil
	}

	if !c.canBeStarted() {
		return errors.New("container cannot be started in current state")
	}

	if err := c.ExecHooks("startContainer"); err != nil {
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	conn, err := net.Dial("unix", filepath.Join(c.State.Bundle, containerSockFilename))
	if err != nil {
		return fmt.Errorf("dial socket: %w", err)
	}

	if err := c.ExecHooks("prestart"); err != nil {
		c.State.Status = specs.StateStopped
		if err := c.hSave(); err != nil {
			return fmt.Errorf("write state file: %w", err)
		}

		// TODO: run DELETE tasks here, then...

		if err := c.ExecHooks("poststop"); err != nil {
			fmt.Println("WARNING: failed to execute poststop hooks")
		}

		return errors.New("failed to run prestart hooks")
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()

	if err := c.ExecHooks("poststart"); err != nil {
		// TODO: how to handle this (log a warning) from start command??
		// FIXME: needs to 'log a warning'
		fmt.Println("WARNING: ", err)
	}

	// FIXME: ?? when process starts, should the process 'replace' the parent container process?

	return nil
}
