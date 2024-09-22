package filesystem

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func MountProc(containerRootfs string) error {
	containerPath := filepath.Join(containerRootfs, "proc")

	if err := os.MkdirAll(
		containerPath,
		os.ModeDir,
	); err != nil {
		return err
	}

	if err := syscall.Mount(
		"proc",
		containerPath,
		"proc",
		uintptr(0),
		"",
	); err != nil {
		return err
	}

	return nil
}

func MountRootfs(containerRootfs string) error {
	if err := syscall.Mount(
		containerRootfs,
		containerRootfs,
		"",
		syscall.MS_BIND|syscall.MS_REC,
		"",
	); err != nil {
		return err
	}

	return nil
}

func DevInSpec(mounts []specs.Mount, dev string) bool {
	for _, mount := range mounts {
		if mount.Destination == dev {
			return true
		}
	}

	return false
}
