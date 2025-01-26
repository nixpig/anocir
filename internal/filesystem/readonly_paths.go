package filesystem

import (
	"golang.org/x/sys/unix"
)

func MountReadonlyPaths(paths []string) error {
	for _, path := range paths {
		initDev := Device{
			Source: path,
			Target: path,
			Fstype: "",
			Flags:  unix.MS_REC | unix.MS_BIND,
			Data:   "",
		}
		if err := initDev.Mount(); err != nil {
			return err
		}

		remountDev := Device{
			Source: path,
			Target: path,
			Fstype: "",
			Flags: unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC |
				unix.MS_BIND | unix.MS_REMOUNT | unix.MS_RDONLY,
			Data: "",
		}
		if err := remountDev.Mount(); err != nil {
			return err
		}
	}

	return nil
}
