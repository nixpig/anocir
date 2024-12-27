package container

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nixpig/brownie/cgroups"
)

func (c *Container) Delete(force bool) error {
	if !force && !c.CanBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state (%s)", c.Status())
	}

	process, err := os.FindProcess(c.PID())
	if err != nil {
		return fmt.Errorf("find container process (%d): %w", c.PID(), err)
	}
	if process != nil {
		process.Signal(syscall.Signal(9))
	}

	if err := cgroups.DeleteV1(c.Spec.Linux.CgroupsPath); err != nil {
		return fmt.Errorf("delete cgroups: %w", err)
	}

	if err := os.RemoveAll(filepath.Join(containerRootDir, c.ID())); err != nil {
		return fmt.Errorf("delete container directory: %w", err)
	}

	if err := c.ExecHooks("poststop"); err != nil {
		fmt.Println("Warning: failed to execute poststop hooks")
	}

	return nil
}
