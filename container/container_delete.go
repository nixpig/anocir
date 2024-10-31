package container

import (
	"fmt"
	"os"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

func (c *Container) Delete() error {
	return deleteContainer(c, false)
}

func (c *Container) ForceDelete() error {
	return deleteContainer(c, true)
}

func deleteContainer(cntr *Container, force bool) error {
	if !force && !cntr.canBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state: %s", cntr.State.Status)
	}

	process, err := os.FindProcess(cntr.State.PID)
	if err != nil {
		return fmt.Errorf("find container process: %w", err)
	}
	if process != nil {
		process.Signal(syscall.Signal(9))
	}

	// TODO: actually do the 'deleting'; rewind all the creation steps

	if err := cntr.ExecHooks("poststop"); err != nil {
		fmt.Println("failed to execute poststop hooks")
		// TODO: log a warning???
	}

	return nil
}
