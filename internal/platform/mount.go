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
// Valid values for flag are MS_SHARED, MS_PRIVATE, MS_SLAVE, MS_BINDABLE.
// Only one flag may be provided at a time.
func SetPropagation(target string, flag uintptr) error {
	// validPropagationFlags := []uintptr{
	// 	unix.MS_SHARED,
	// 	unix.MS_PRIVATE,
	// 	unix.MS_SLAVE,
	// 	unix.MS_UNBINDABLE,
	// }
	//
	// if !slices.Contains(validPropagationFlags, flag) {
	// 	return fmt.Errorf("invalid propagation flag: 0x%x", flag)
	// }

	return mount("", target, "", flag, "")
}
