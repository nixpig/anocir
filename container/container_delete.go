package container

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/rs/zerolog"
)

func (c *Container) Delete(force bool, log *zerolog.Logger) error {
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
		log.Warn().Err(err).Msg("failed to execute poststop hooks")
		fmt.Println("WARNING: failed to execute poststop hooks")
	}

	if err := os.RemoveAll(
		filepath.Join(containerRootDir, c.ID()),
	); err != nil {
		return err
	}

	return nil
}
