package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// MountProc mounts the /proc filesystem inside the given containerRootfs.
func MountProc(containerRootfs string) error {
	containerProc := filepath.Join(containerRootfs, "proc")
	if err := os.MkdirAll(containerProc, 0o666); err != nil {
		return fmt.Errorf("create proc dir: %w", err)
	}

	if err := MountFilesystem(
		"proc",
		containerProc,
		"proc",
		uintptr(0),
		"",
	); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	return nil
}
