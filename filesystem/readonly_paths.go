package filesystem

import "syscall"

func MountReadonlyPaths(paths []string) error {
	for _, path := range paths {
		if err := MountDevice(Device{
			Source: path,
			Target: path,
			Fstype: "",
			Flags:  syscall.MS_REC | syscall.MS_BIND,
			Data:   "",
		}); err != nil {
			return err
		}

		if err := MountDevice(Device{
			Source: path,
			Target: path,
			Fstype: "",
			Flags: syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_NOEXEC |
				syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_RDONLY,
			Data: "",
		}); err != nil {
			return err
		}
	}

	return nil
}
