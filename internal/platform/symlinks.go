package platform

import (
	"errors"
	"os"
	"path/filepath"
)

// defaultSymlinks are the symlinks required by all containers to ensure
// common device paths work correctly.
var defaultSymlinks = map[string]string{
	"/proc/self/fd":   "dev/fd",
	"/proc/self/fd/0": "dev/stdin",
	"/proc/self/fd/1": "dev/stdout",
	"/proc/self/fd/2": "dev/stderr",
	"pts/ptmx":        "dev/ptmx",
}

// CreateDefaultSymlinks creates the default symlinks inside the
// containerRootfs.
func CreateDefaultSymlinks(containerRootfs string) error {
	return createSymlinks(defaultSymlinks, containerRootfs)
}

func createSymlinks(symlinks map[string]string, rootfs string) error {
	for src, dest := range symlinks {
		destPath := filepath.Join(rootfs, dest)

		if target, err := os.Readlink(destPath); err == nil {
			if target == src {
				continue
			}
			if err := os.Remove(destPath); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			if err := os.Remove(destPath); err != nil {
				return err
			}
		}

		if err := os.Symlink(src, destPath); err != nil {
			return err
		}
	}

	return nil
}
