package filesystem

import (
	"os"
	"path/filepath"
)

var DefaultSymlinks = map[string]string{
	"/proc/self/fd":   "dev/fd",
	"/proc/self/fd/0": "dev/stdin",
	"/proc/self/fd/1": "dev/stdout",
	"/proc/self/fd/2": "dev/stderr",
	"pts/ptmx":        "dev/ptmx",
}

func CreateSymlinks(symlinks map[string]string, rootfs string) error {
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
