package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// lockFilename is the filename of the lockfile used to synchronise access
// to container operations.
const lockFilename = "c.lock"

// ErrOperationInProgress is returned when the container is locked by another
// operation.
var ErrOperationInProgress = errors.New("operation already in progress")

func (c *Container) Lock() error {
	lockPath := filepath.Join(c.RootDir, c.State.ID, lockFilename)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()

		if err == unix.EWOULDBLOCK {
			return ErrOperationInProgress
		}

		return fmt.Errorf("acquire file lock: %w", err)
	}

	c.lockFile = f
	return nil
}

func (c *Container) Unlock() error {
	if c.lockFile == nil {
		return nil
	}

	defer c.lockFile.Close()
	return unix.Flock(int(c.lockFile.Fd()), unix.LOCK_UN)
}

// DoWithLock acquires an exclusive lock on the container, refreshes the state,
// and executes the given fn, finally releasing the lock.
func (c *Container) DoWithLock(fn func(*Container) error) error {
	if err := c.Lock(); err != nil {
		return fmt.Errorf("lock access to container: %w", err)
	}
	defer c.Unlock()

	if err := c.reloadState(); err != nil {
		return fmt.Errorf("reload container state: %w", err)
	}

	return fn(c)
}

func (c *Container) reloadState() error {
	s, err := os.ReadFile(c.stateFilepath())
	if err != nil {
		return fmt.Errorf("read state file: %w", err)
	}

	if err := json.Unmarshal(s, c.State); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	return nil
}
