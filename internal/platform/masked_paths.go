package platform

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// MountMaskedPaths mounts over the specified paths to mask them. If the path
// is not found, it's skipped.
func MountMaskedPaths(paths []string) error {
	for _, p := range paths {
		f, err := os.Stat(p)
		if err != nil {
			// if it's not there, there's nothing to mask; skip it
			continue
		}

		if f.IsDir() {
			if err := MountFilesystem(
				"tmpfs",
				p,
				"tmpfs",
				unix.MS_RDONLY,
				"",
			); err != nil {
				return fmt.Errorf("mount tmpfs masked path: %w", err)
			}
		} else {
			if err := BindMount("/dev/null", p, true); err != nil {
				return fmt.Errorf("bind mount masked path: %w", err)
			}
		}
	}

	return nil
}
