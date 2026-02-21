package platform

import (
	"fmt"
	"os"
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

// CreateDefaultSymlinks creates the default symlinks inside the containerRootfs.
func CreateDefaultSymlinks(containerRootfs string) error {
	return createSymlinks(defaultSymlinks, containerRootfs)
}

func createSymlinks(symlinks map[string]string, rootfs string) error {
	root, err := os.OpenRoot(rootfs)
	if err != nil {
		return fmt.Errorf("open rootfs: %w", err)
	}
	defer root.Close()

	for src, dest := range symlinks {
		_ = root.Remove(dest)
		if err := root.Symlink(src, dest); err != nil {
			return fmt.Errorf("create symlink %s -> %s: %w", dest, src, err)
		}
	}

	return nil
}
