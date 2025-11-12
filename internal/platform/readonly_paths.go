package platform

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// MountReadonlyPaths bind mounts and then remounts the specified paths as read-only.
func MountReadonlyPaths(paths []string) error {
	for _, p := range paths {
		if err := BindMount(p, p, true); err != nil {
			return fmt.Errorf("initial bind mount readonly paths: %w", err)
		}

		flags := unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC |
			unix.MS_BIND | unix.MS_REMOUNT | unix.MS_RDONLY

		if err := Remount(p, uintptr(flags)); err != nil {
			return fmt.Errorf("remount readonly paths: %w", err)
		}
	}

	return nil
}
