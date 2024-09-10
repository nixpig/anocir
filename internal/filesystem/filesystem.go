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
			// relativePath,
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

func MountDev(containerRootfs string) error {
	if err := os.MkdirAll(
		filepath.Join(containerRootfs, "dev"), os.ModeDir,
	); err != nil {
		return err
	}

	if err := syscall.Mount(
		"tmpfs",
		filepath.Join(containerRootfs, "dev"),
		"tmpfs",
		syscall.MS_NOSUID|syscall.MS_STRICTATIME,
		"mode=755",
	); err != nil {
		return err
	}

	return nil
}
