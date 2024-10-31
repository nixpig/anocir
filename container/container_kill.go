package container

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Kill(sig syscall.Signal) error {
	if !c.canBeKilled() {
		return errors.New("container cannot be killed in current state")
	}

	if err := syscall.Kill(c.State.PID, sig); err != nil {
		return fmt.Errorf("failed to execute kill syscall: %w", err)
	}

	c.State.Status = specs.StateStopped
	if err := c.hSave(); err != nil {
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	return nil
}
