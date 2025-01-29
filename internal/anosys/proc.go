package anosys

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func MountProc(containerRootfs string) error {
	containerProc := filepath.Join(containerRootfs, "proc")
	if err := os.MkdirAll(containerProc, 0666); err != nil {
		return fmt.Errorf("create proc dir: %w", err)
	}

	if err := syscall.Mount(
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
