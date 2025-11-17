package platform

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func mount(source, target, fstype string, flags uintptr, data string) error {
	if err := unix.Mount(source, target, fstype, flags, data); err != nil {
		return fmt.Errorf(
			"mount %s to %s (type=%s, flags=%#x): %w",
			source, target, fstype, flags, err,
		)
	}

	return nil
}

func BindMount(source, target string, rec bool) error {
	flags := unix.MS_BIND

	if rec {
		flags |= unix.MS_REC
	}

	return mount(source, target, "", uintptr(flags), "")
}

func Remount(target string, flags uintptr) error {
	return mount("", target, "", unix.MS_REMOUNT|flags, "")
}

func MountFilesystem(
	source, target, fstype string,
	flags uintptr,
	data string,
) error {
	return mount(source, target, fstype, flags, data)
}

func SetPropagation(target string, flags uintptr) error {
	return mount("", target, "", flags, "")
}
