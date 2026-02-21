package platform

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// MountReadonlyPaths bind mounts and then remounts the given paths as
// read-only.
func MountReadonlyPaths(paths []string) error {
	flags := unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC |
		unix.MS_BIND | unix.MS_REMOUNT | unix.MS_RDONLY

	for _, p := range paths {
		if err := BindMount(p, p, true); err != nil {
			return fmt.Errorf("initial bind mount readonly paths: %w", err)
		}

		if err := Remount(p, uintptr(flags)); err != nil {
			return fmt.Errorf("remount readonly paths: %w", err)
		}
	}

	return nil
}
