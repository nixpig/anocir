package anosys

import (
	"os"
	"path/filepath"
)

var defaultSymlinks = map[string]string{
	"/proc/self/fd":   "dev/fd",
	"/proc/self/fd/0": "dev/stdin",
	"/proc/self/fd/1": "dev/stdout",
	"/proc/self/fd/2": "dev/stderr",
	"pts/ptmx":        "dev/ptmx",
}

// CreateDefaultSymlinks creates the default symlinks inside the container's
// root filesystem.
func CreateDefaultSymlinks(rootfs string) error {
	return createSymlinks(defaultSymlinks, rootfs)
}

func createSymlinks(symlinks map[string]string, rootfs string) error {
	for src, dest := range symlinks {
		if err := os.Symlink(
			src,
			filepath.Join(rootfs, dest),
		); err != nil {
			return err
		}
	}

	return nil
}
