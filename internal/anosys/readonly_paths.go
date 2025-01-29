package anosys

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func MountReadonlyPaths(paths []string) error {
	for _, path := range paths {
		if err := syscall.Mount(
			path,
			path,
			"",
			unix.MS_REC|unix.MS_BIND,
			"",
		); err != nil {
			return fmt.Errorf("initial bind mount ro paths: %w", err)
		}

		if err := syscall.Mount(
			path,
			path,
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
