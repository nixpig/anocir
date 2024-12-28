package filesystem

import (
	"os"

	"golang.org/x/sys/unix"
)

func MountMaskedPaths(paths []string) error {
	for _, path := range paths {
		f, err := os.Stat(path)
		if err != nil {
			continue
		}

		if f.IsDir() {
			if err := MountDevice(Device{
				Source: "tmpfs",
				Target: path,
				Fstype: "tmpfs",
				Flags:  unix.MS_RDONLY,
				Data:   "",
			}); err != nil {
				return err
			}
		} else {
			if err := MountDevice(Device{
				Source: "/dev/null",
				Target: path,
				Fstype: "bind",
				Flags:  unix.MS_BIND,
				Data:   "",
			}); err != nil {
				return err
			}
		}

	}

	return nil
}
