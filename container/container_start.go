package container

import (
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
		// if there's no process, there's nothing to do; return silently
		return nil
	}

	if !c.CanBeStarted() {
		return fmt.Errorf("container cannot be started in current state (%s)", c.Status())
	}

	if err := c.ExecHooks("prestart"); err != nil {
		// TODO: rollback and delete (?), then...
		if err := c.ExecHooks("poststop"); err != nil {
			fmt.Println("Warning: failed to execute poststop hooks: ", err)
		}

		return fmt.Errorf("execute prestart hooks: %w", err)
	}

	// send "start"
	conn, err := net.Dial("unix", filepath.Join(containerRootDir, c.ID(), containerSockFilename))
	if err != nil {
		return fmt.Errorf("dial container socket: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		return fmt.Errorf("send 'start' to container: %w", err)
	}
	defer conn.Close()

	c.SetStatus(specs.StateRunning)
	if err := c.Save(); err != nil {
		return fmt.Errorf("save running container state: %w", err)
	}

	if err := c.ExecHooks("poststart"); err != nil {
		fmt.Println("Warning: ", err)
	}

	return nil
}
