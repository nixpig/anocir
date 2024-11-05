package container

import (
	"fmt"
	"os"
	"syscall"
)

func (c *Container) Delete(force bool) error {
	if !force && !c.CanBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state: %s", c.Status())
	}

	process, err := os.FindProcess(c.PID())
	if err != nil {
		return fmt.Errorf("find container process: %w", err)
	}
	if process != nil {
		process.Signal(syscall.Signal(9))
	}

	// TODO: actually do the 'deleting'; rewind all the creation steps

	if err := c.ExecHooks("poststop"); err != nil {
		fmt.Println("failed to execute poststop hooks")
		// TODO: log a warning???
	}

	return nil
}
