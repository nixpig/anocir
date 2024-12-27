package container

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/cgroups"
	"golang.org/x/sys/unix"
)

func (c *Container) Delete(force bool) error {
	if !force && !c.CanBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state (%s)", c.Status())
	}

	if err := c.Kill(unix.SIGKILL); err != nil {
		return fmt.Errorf("send sigkill to container: %w", err)
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
