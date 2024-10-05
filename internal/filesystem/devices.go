package filesystem

import (
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
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

var DefaultDevices = []specs.LinuxDevice{
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
