package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func MountProc(containerRootfs string) error {
	if err := os.MkdirAll(
		filepath.Join(containerRootfs, "proc"),
		os.ModeDir,
	); err != nil {
		return fmt.Errorf("make proc dir: %w", err)
	}

	if err := syscall.Mount(
		"proc",
		filepath.Join(containerRootfs, "proc"),
		"proc",
		uintptr(0),
		"",
	); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	return nil
}

func unmountProc() error {
	return syscall.Unmount("proc", 0)
}
