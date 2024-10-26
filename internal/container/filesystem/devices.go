package filesystem

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

type Device struct {
	Source string
	Target string
	Fstype string
	Flags  uintptr
	Data   string
}

var (
	defaultFileMode        = os.FileMode(0666)
	defaultUID      uint32 = 0
	defaultGID      uint32 = 0
)

var (
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

func mountDefaultDevices(rootfs string) error {
	return mountDevices(defaultDevices, rootfs)
}

func mountSpecDevices(devices []specs.LinuxDevice, rootfs string, log *zerolog.Logger) error {
	for _, dev := range devices {
		log.Info().Any("dev", dev).Msg("setup device")

		var absPath string
		if strings.Index(dev.Path, "/") == 0 {
			relPath := strings.TrimPrefix(dev.Path, "/")
			absPath = filepath.Join(rootfs, relPath)
		} else {
			absPath = filepath.Join(rootfs, dev.Path)
		}

		log.Info().
			Str("path", absPath).
			Uint32("filemode", uint32(*dev.FileMode)).
			Int("dev", int(unix.Mkdev(uint32(dev.Major), uint32(dev.Minor)))).
			Msg("make node")
		if err := unix.Mknod(
			absPath,
			uint32(*dev.FileMode),
			int(unix.Mkdev(uint32(dev.Major), uint32(dev.Minor))),
		); err != nil {
			return err
		}

		log.Info().
			Str("path", absPath).
			Int("uid", int(*dev.UID)).
			Int("gid", int(*dev.GID)).
			Msg("chown")
		if err := os.Chown(
			absPath,
			int(*dev.UID),
			int(*dev.GID),
		); err != nil {
			return err
		}
	}

	return nil
}
