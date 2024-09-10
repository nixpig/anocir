package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nixpig/brownie/pkg/config"
)

var (
	defaultFileMode uint32 = 066
	defaultUid      uint32 = 0
	defaultGid      uint32 = 0
)

var defaultDevices = []config.Device{
	{
		Type:     config.CharDevice,
		Path:     "/dev/null",
		Major:    1,
		Minor:    3,
		FileMode: &defaultFileMode,
		Uid:      &defaultUid,
		Gid:      &defaultGid,
	},
	{
		Type:     config.CharDevice,
		Path:     "/dev/full",
		Major:    1,
		Minor:    7,
		FileMode: &defaultFileMode,
		Uid:      &defaultUid,
		Gid:      &defaultGid,
	},
	{
		Type:     config.CharDevice,
		Path:     "/dev/zero",
		Major:    1,
		Minor:    5,
		FileMode: &defaultFileMode,
		Uid:      &defaultUid,
		Gid:      &defaultGid,
	},
	{
		Type:     config.CharDevice,
		Path:     "/dev/random",
		Major:    1,
		Minor:    8,
		FileMode: &defaultFileMode,
		Uid:      &defaultUid,
		Gid:      &defaultGid,
	},
	{
		Type:     config.CharDevice,
		Path:     "/dev/urandom",
		Major:    1,
		Minor:    9,
		FileMode: &defaultFileMode,
		Uid:      &defaultUid,
		Gid:      &defaultGid,
	},
	{
		Type:     config.CharDevice,
		Path:     "/dev/tty",
		Major:    5,
		Minor:    0,
		FileMode: &defaultFileMode,
		Uid:      &defaultUid,
		Gid:      &defaultGid,
	},
}

func MountDefaultDevices(containerRootfs string) error {
	for _, dev := range defaultDevices {
		relativePath := strings.TrimLeft(dev.Path, "/")
		containerPath := filepath.Join(containerRootfs, relativePath)

		if err := os.MkdirAll(containerPath, fs.FileMode(*dev.FileMode)); err != nil {
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
		return fmt.Errorf("ensure proc destination exists: %w", err)
	}

	if err := syscall.Mount(
		"proc",
		containerPath,
		"proc",
		uintptr(0),
		"",
	); err != nil {
		return fmt.Errorf("mount proc: %w", err)
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
		return fmt.Errorf("mount rootfs: %w", err)
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
