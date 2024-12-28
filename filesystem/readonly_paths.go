package filesystem

import (
	"golang.org/x/sys/unix"
)

func MountReadonlyPaths(paths []string) error {
	for _, path := range paths {
		if err := MountDevice(Device{
			Source: path,
			Target: path,
			Fstype: "",
			Flags:  unix.MS_REC | unix.MS_BIND,
			Data:   "",
		}); err != nil {
			return err
		}

		if err := MountDevice(Device{
			Source: path,
			Target: path,
			Fstype: "",
			Flags: unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC |
				unix.MS_BIND | unix.MS_REMOUNT | unix.MS_RDONLY,
			Data: "",
		}); err != nil {
			return err
		}
	}

	return nil
}
