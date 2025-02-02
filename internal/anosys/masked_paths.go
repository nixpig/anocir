package anosys

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func MountMaskedPaths(paths []string) error {
	for _, p := range paths {
		f, err := os.Stat(p)
		if err != nil {
			// if it's not there, there's nothing to mask; skip it
			continue
		}

		if f.IsDir() {
			if err := syscall.Mount(
				"tmpfs",
				p,
				"tmpfs",
				unix.MS_RDONLY,
				"",
			); err != nil {
				return fmt.Errorf("mount tmpfs masked path: %w", err)
			}
		} else {
			if err := syscall.Mount(
				"/dev/null",
				p,
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
