package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func PivotRootfs(containerRootfs string) error {
	oldroot := filepath.Join(containerRootfs, "oldroot")

	if err := os.MkdirAll(oldroot, 0700); err != nil {
		return fmt.Errorf("make old root dir: %w", err)
	}

	if err := syscall.PivotRoot(containerRootfs, oldroot); err != nil {
		return fmt.Errorf("pivot to new root: %w", err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir to new root: %w", err)
	}

	if err := syscall.Unmount("oldroot", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root: %w", err)
	}

	if err := os.RemoveAll("oldroot"); err != nil {
		return fmt.Errorf("remove old root: %w", err)
	}

	return nil
}
