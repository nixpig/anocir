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
		c.SetStatus(specs.StateStopped)
		if err := c.Save(); err != nil {
			return err
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

	if err := c.ExecHooks("prestart"); err != nil {
		// TODO: run DELETE tasks here, then...
		if err := c.ExecHooks("poststop"); err != nil {
			fmt.Println("Warning: failed to execute poststop hooks")
		}

		fmt.Println("Warning: failed to execute prestart hooks")
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()

	c.SetStatus(specs.StateRunning)
	if err := c.Save(); err != nil {
		return fmt.Errorf("save host container state: %w", err)
	}
	// FIXME: do these need to move up before the cmd.Wait call??
	if err := c.ExecHooks("poststart"); err != nil {
		fmt.Println("Warning: ", err)
	}

	return nil
}
