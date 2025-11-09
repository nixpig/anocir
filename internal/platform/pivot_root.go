package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

const oldroot = ".oldroot"

// PivotRoot changes the root filesystem of the calling process to the
// specified container rootfs path.
func PivotRoot(containerRootfs string) error {
	if err := os.MkdirAll(
		filepath.Join(containerRootfs, oldroot),
		0o700,
	); err != nil {
		return fmt.Errorf("make old root dir: %w", err)
	}

	if err := syscall.PivotRoot(
		containerRootfs,
		filepath.Join(containerRootfs, oldroot),
	); err != nil {
		return fmt.Errorf("pivot to new root: %w", err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir to new root: %w", err)
	}

	if err := syscall.Unmount(oldroot, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root: %w", err)
	}

	if err := os.RemoveAll(oldroot); err != nil {
		return fmt.Errorf("remove old root: %w", err)
	}

	return nil
}
