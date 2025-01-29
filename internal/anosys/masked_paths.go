package anosys

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func MountMaskedPaths(paths []string) error {
	for _, path := range paths {
		f, err := os.Stat(path)
		if err != nil {
			continue
		}

		if f.IsDir() {
			if err := syscall.Mount(
				"tmpfs",
				path,
				"tmpfs",
				unix.MS_RDONLY,
				"",
			); err != nil {
				return fmt.Errorf("mount tmpfs masked path: %w", err)
			}
		} else {
			if err := syscall.Mount(
				"/dev/null",
				path,
				"bind",
				unix.MS_BIND,
				"",
			); err != nil {
				return fmt.Errorf("bind mount masked path: %w", err)
			}
		}
	}

	return nil
}
