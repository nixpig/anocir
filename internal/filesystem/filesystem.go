package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

var (
	defaultFileMode        = os.FileMode(0066)
	defaultUID      uint32 = 0
	defaultGID      uint32 = 0

	AllDevices           = "a"
	BlockDevice          = "b"
	CharDevice           = "c"
	UnbufferedCharDevice = "u"
	FifoDevice           = "p"
)

var defaultDevices = []specs.LinuxDevice{
	{
		Path:     "/dev/null",
		Type:     CharDevice,
		Major:    1,
		Minor:    3,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     CharDevice,
		Path:     "/dev/full",
		Major:    1,
		Minor:    7,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     CharDevice,
		Path:     "/dev/zero",
		Major:    1,
		Minor:    5,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	// {
	// 	Type:     CharDevice,
	// 	Path:     "/dev/random",
	// 	Major:    1,
	// 	Minor:    8,
	// 	FileMode: &defaultFileMode,
	// 	UID:      &defaultUID,
	// 	GID:      &defaultGID,
	// },
	{
		Type:     CharDevice,
		Path:     "/dev/urandom",
		Major:    1,
		Minor:    9,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     CharDevice,
		Path:     "/dev/tty",
		Major:    5,
		Minor:    0,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
}

func MountDefaultDevices(containerRootfs string) error {
	for _, dev := range defaultDevices {
		relativePath := strings.TrimLeft(dev.Path, "/")
		containerPath := filepath.Join(containerRootfs, relativePath)

		if err := os.MkdirAll(containerPath, *dev.FileMode); err != nil {
			return fmt.Errorf("ensure dev destination exists: %w", err)
		}

		if err := syscall.Mount(
			"tmpfs",
			containerPath,
			"tmpfs",
			uintptr(0),
			"",
		); err != nil {
			return fmt.Errorf("mount device: %w", err)
		}
	}

	return nil
}

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

func UnmountProc() error {
	return syscall.Unmount("proc", 0)
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

func PivotRootfs(containerRootfs string) error {
	oldroot := filepath.Join(containerRootfs, "oldroot")

	if err := os.MkdirAll(oldroot, 0700); err != nil {
		return err
	}

	if err := syscall.PivotRoot(containerRootfs, oldroot); err != nil {
		return err
	}

	if err := os.Chdir("/"); err != nil {
		return err
	}

	if err := syscall.Unmount("oldroot", syscall.MNT_DETACH); err != nil {
		return err
	}

	if err := os.RemoveAll("oldroot"); err != nil {
		return err
	}

	return nil
}
