package platform

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// PivotRoot changes the root filesystem of the container process to the given containerRootfs path.
func PivotRoot(containerRootfs string) error {
	// Open fd to old root so we can fchdir back after the pivot.
	oldroot, err := unix.Open("/", unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open old root fd: %w", err)
	}
	defer unix.Close(oldroot)

	newroot, err := unix.Open(containerRootfs, unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open new root fd: %w", err)
	}
	defer unix.Close(newroot)

	if err := unix.Fchdir(newroot); err != nil {
		return fmt.Errorf("fchdir to new root: %w", err)
	}

	if err := unix.PivotRoot(".", "."); err != nil {
		return fmt.Errorf("pivot to new root: %w", err)
	}

	if err := unix.Fchdir(oldroot); err != nil {
		return fmt.Errorf("fchdir to old root: %w", err)
	}

	// Make oldroot rslave to prevent unmounts from propagating to the host.
	// After pivot_root, the old root is at "." (current directory).
	if err := unix.Mount("", ".", "", unix.MS_SLAVE|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("make old root slave: %w", err)
	}

	if err := unix.Unmount(".", unix.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root: %w", err)
	}

	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("chdir to new root: %w", err)
	}

	return nil
}
