package platform

import (
	"fmt"
	"slices"

	"golang.org/x/sys/unix"
)

var validPropagationFlags = []uintptr{
	0,
	unix.MS_SHARED,
	unix.MS_PRIVATE,
	unix.MS_SLAVE,
	unix.MS_UNBINDABLE,
}

func mount(source, target, fstype string, flags uintptr, data string) error {
	if err := unix.Mount(source, target, fstype, flags, data); err != nil {
		return fmt.Errorf(
			"mount %s to %s (type=%s, flags=%#x): %w",
			source, target, fstype, flags, err,
		)
	}

	return nil
}

// BindMount bind mounts the source to target. If rec is true, then it performs
// the mount recursively.
func BindMount(source, target string, rec bool) error {
	flags := unix.MS_BIND

	if rec {
		flags |= unix.MS_REC
	}

	return mount(source, target, "", uintptr(flags), "")
}

// Remount changes the mount flags of the given target by remounting and
// applying the given flags.
func Remount(target string, flags uintptr) error {
	return mount("", target, "", unix.MS_REMOUNT|flags, "")
}

// MountFilesystem mounts a filesystem of type fstype from source to target,
// applying the given flags.
func MountFilesystem(
	source, target, fstype string,
	flags uintptr,
	data string,
) error {
	return mount(source, target, fstype, flags, data)
}

// SetPropagation sets the propagation type for the mount at the given target.
// Valid values for flag are MS_SHARED, MS_PRIVATE, MS_SLAVE, MS_BINDABLE and
// only one flag may be provided. The MS_REC modifier can be OR'd with any
// propagation type to make it recursive.
func SetPropagation(target string, flag uintptr) error {
	if !validatePropagationFlag(flag) {
		return fmt.Errorf("invalid propagation flag: 0x%x", flag)
	}

	return mount("", target, "", flag, "")
}

// TODO: Add unit tests.
func validatePropagationFlag(flag uintptr) bool {
	baseFlag := flag &^ unix.MS_REC

	return slices.Contains(validPropagationFlags, baseFlag)
}
