package platform

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// MountReadonlyPaths bind mounts and then remounts the specified paths as read-only.
func MountReadonlyPaths(paths []string) error {
	for _, p := range paths {
		if err := syscall.Mount(
			p,
			p,
			"",
			unix.MS_REC|unix.MS_BIND,
			"",
		); err != nil {
			return fmt.Errorf("initial bind mount ro paths: %w", err)
		}

		if err := syscall.Mount(
			p,
			p,
			"",
			unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC|
				unix.MS_BIND|unix.MS_REMOUNT|unix.MS_RDONLY,
			"",
		); err != nil {
			return fmt.Errorf("remount ro paths: %w", err)
		}
	}

	return nil
}
