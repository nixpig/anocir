package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var (
	defaultFileMode        = os.FileMode(0o666)
	defaultUID      uint32 = 0
	defaultGID      uint32 = 0

	allDevices           = "a"
	blockDevice          = "b"
	charDevice           = "c"
	unbufferedCharDevice = "u"
	fifoDevice           = "p"
)

// deviceType maps device type strings to their corresponding kernel values.
var deviceType = map[string]uint32{
	"b": unix.S_IFBLK,
	"c": unix.S_IFCHR,
	"s": unix.S_IFSOCK,
	"p": unix.S_IFIFO,
}

var defaultDevices = []specs.LinuxDevice{
	{
		Type:     charDevice,
		Path:     "/dev/null",
		Major:    1,
		Minor:    3,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     charDevice,
		Path:     "/dev/zero",
		Major:    1,
		Minor:    5,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     charDevice,
		Path:     "/dev/full",
		Major:    1,
		Minor:    7,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     charDevice,
		Path:     "/dev/random",
		Major:    1,
		Minor:    8,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     charDevice,
		Path:     "/dev/urandom",
		Major:    1,
		Minor:    9,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
	{
		Type:     charDevice,
		Path:     "/dev/tty",
		Major:    5,
		Minor:    0,
		FileMode: &defaultFileMode,
		UID:      &defaultUID,
		GID:      &defaultGID,
	},
}

// MountDefaultDevices mounts the default set of devices into the containers
// root filesystem at the given containerRootfs.
func MountDefaultDevices(containerRootfs string) error {
	root, err := os.OpenRoot(containerRootfs)
	if err != nil {
		return fmt.Errorf("open container rootfs: %w", err)
	}
	defer root.Close()

	for _, d := range defaultDevices {
		relPath := strings.TrimPrefix(d.Path, "/")

		f, err := root.Create(relPath)
		if err != nil && !os.IsExist(err) {
			return err
		}
		if f != nil {
			f.Close()
		}

		absPath := filepath.Join(root.Name(), relPath)
		if err := BindMount(d.Path, absPath, false); err != nil {
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

		if err := unix.Chmod(absPath, uint32(*d.FileMode)); err != nil {
			return err
		}

		if d.UID != nil && d.GID != nil {
			if err := os.Chown(absPath, int(*d.UID), int(*d.GID)); err != nil {
				return err
			}
		}
	}

	return nil
}
