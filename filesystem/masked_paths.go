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

		var dev Device

		if f.IsDir() {
			dev = Device{
				Source: "tmpfs",
				Target: path,
				Fstype: "tmpfs",
				Flags:  unix.MS_RDONLY,
				Data:   "",
			}
		} else {
			dev = Device{
				Source: "/dev/null",
				Target: path,
				Fstype: "bind",
				Flags:  unix.MS_BIND,
				Data:   "",
			}
		}

		if err := dev.Mount(); err != nil {
			return err
		}

	}

	return nil
}
