package platform

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// PivotRoot changes the root filesystem of the calling process to the
// specified container rootfs path.
func PivotRoot(containerRootfs string) error {
	if err := unix.PivotRoot(containerRootfs, containerRootfs); err != nil {
		return fmt.Errorf("pivot to new root: %w", err)
	}

	if err := unix.Unmount("/", unix.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root: %w", err)
	}

	return nil
}
