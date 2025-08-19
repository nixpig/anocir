package anosys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var (
	AllDevices           = "a"
	BlockDevice          = "b"
	CharDevice           = "c"
	UnbufferedCharDevice = "u"
	FifoDevice           = "p"
)

var deviceType = map[string]uint32{
	"b": unix.S_IFBLK,
	"c": unix.S_IFCHR,
	"s": unix.S_IFSOCK,
	"p": unix.S_IFIFO,
}

var (
	defaultFileMode        = os.FileMode(0666)
	defaultUID      uint32 = 0
	defaultGID      uint32 = 0
)

var defaultDevices = []specs.LinuxDevice{
	{
		Type:     CharDevice,
		Path:     "/dev/null",
		Major:    1,
		Minor:    3,
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
		Path:     "/dev/random",
		Major:    1,
		Minor:    8,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
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

// MountDefaultDevices mounts the default set of devices into the container's
// root filesystem.
func MountDefaultDevices(rootfs string) error {
	for _, d := range defaultDevices {
		absPath := filepath.Join(rootfs, strings.TrimPrefix(d.Path, "/"))

		f, err := os.Create(absPath)
		if err != nil && !os.IsExist(err) {
			return err
		}
		f.Close()

		if err := syscall.Mount(
			d.Path,
			absPath,
			"bind",
			unix.MS_BIND,
			"",
		); err != nil {
			return fmt.Errorf("bind mount device: %w", err)
		}
	}

	return nil
}

// CreateDeviceNodes creates device nodes in the container's root filesystem
// based on the provided LinuxDevice specs.
func CreateDeviceNodes(devices []specs.LinuxDevice, rootfs string) error {
	for _, d := range devices {
		absPath := filepath.Join(rootfs, strings.TrimPrefix(d.Path, "/"))

		if err := unix.Mknod(
			absPath,
			deviceType[d.Type],
			int(unix.Mkdev(uint32(d.Major), uint32(d.Minor))),
		); err != nil {
			return err
		}

		if err := syscall.Chmod(absPath, uint32(*d.FileMode)); err != nil {
			return err
		}

		if d.UID != nil && d.GID != nil {
			if err := os.Chown(
				absPath,
				int(*d.UID),
				int(*d.GID),
			); err != nil {
				return err
			}
		}
	}

	return nil
}
