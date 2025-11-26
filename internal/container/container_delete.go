package container

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// Delete removes the Container from the system. If force is true then it will
// delete the Container, regardless of the its state.
func (c *Container) Delete(force bool) error {
	if !force && !c.canBeDeleted() {
		return fmt.Errorf(
			"container cannot be deleted in current state (%s) try using '--force'",
			c.State.Status,
		)
	}

	if c.spec.Linux.Resources != nil {
		if err := platform.DeleteCGroups(c.State, c.spec); err != nil {
			return fmt.Errorf("delete cgroups: %w", err)
		}
	} else if c.State.Pid != 0 {
		if err := unix.Kill(
			c.State.Pid,
			unix.SIGKILL,
		); err != nil && !errors.Is(err, unix.ESRCH) {
			return fmt.Errorf("send kill signal to process: %w", err)
		}

		unix.Wait4(c.State.Pid, nil, 0, nil)
	}

	// TODO: Review whether need to remove pidfile.

	if err := os.RemoveAll(
		filepath.Join(c.RootDir, c.State.ID),
	); err != nil {
		return fmt.Errorf("delete container directory: %w", err)
	}

	if err := c.execHooks(LifecyclePoststop); err != nil {
		fmt.Printf("Warning: failed to exec poststop hooks: %s\n", err)
	}

	return nil
}

func (c *Container) canBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}
