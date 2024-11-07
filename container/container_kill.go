package container

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Kill(sig syscall.Signal) error {
	if !c.CanBeKilled() {
		return errors.New("container cannot be killed in current state")
	}

	if err := syscall.Kill(c.PID(), sig); err != nil {
		return fmt.Errorf("failed to execute kill syscall (process: %d): %w", c.PID(), err)
	}

	c.SetStatus(specs.StateStopped)
	if err := c.HSave(); err != nil {
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	return nil
}
