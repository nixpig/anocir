package platform

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type mountOption struct {
	flag      uintptr
	recursive bool
	invert    bool
}

var mountOptions = map[string]mountOption{
	"async": {
		invert:    true,
		recursive: false,
		flag:      unix.MS_SYNCHRONOUS,
	},
	"atime":      {invert: true, recursive: true, flag: unix.MS_NOATIME},
	"bind":       {invert: false, recursive: false, flag: unix.MS_BIND},
	"defaults":   {invert: false, recursive: false, flag: 0},
	"dev":        {invert: true, recursive: false, flag: unix.MS_NODEV},
	"diratime":   {invert: true, recursive: false, flag: unix.MS_NODIRATIME},
	"dirsync":    {invert: false, recursive: false, flag: unix.MS_DIRSYNC},
	"exec":       {invert: true, recursive: false, flag: unix.MS_NOEXEC},
	"iversion":   {invert: false, recursive: false, flag: unix.MS_I_VERSION},
	"lazytime":   {invert: false, recursive: false, flag: unix.MS_LAZYTIME},
	"loud":       {invert: true, recursive: false, flag: unix.MS_SILENT},
	"noatime":    {invert: false, recursive: true, flag: unix.MS_NOATIME},
	"nodev":      {invert: false, recursive: true, flag: unix.MS_NODEV},
	"nodiratime": {invert: false, recursive: true, flag: unix.MS_NODIRATIME},
	"noexec":     {invert: false, recursive: true, flag: unix.MS_NOEXEC},
	"noiversion": {invert: true, recursive: false, flag: unix.MS_I_VERSION},
	"nolazytime": {invert: true, recursive: false, flag: unix.MS_LAZYTIME},
	"norelatime": {invert: true, recursive: false, flag: unix.MS_RELATIME},
	"nostrictatime": {
		invert:    true,
		recursive: false,
		flag:      unix.MS_STRICTATIME,
	},
	"nosuid": {invert: false, recursive: true, flag: unix.MS_NOSUID},
	"nosymfollow": {
		invert:    false,
		recursive: true,
		flag:      unix.MS_NOSYMFOLLOW,
	},
	"private":  {invert: false, recursive: false, flag: unix.MS_PRIVATE},
	"rbind":    {invert: false, recursive: true, flag: unix.MS_BIND},
	"relatime": {invert: false, recursive: true, flag: unix.MS_RELATIME},
	"remount":  {invert: false, recursive: false, flag: unix.MS_REMOUNT},
	"ro":       {invert: false, recursive: true, flag: unix.MS_RDONLY},
	"rprivate": {invert: false, recursive: true, flag: unix.MS_PRIVATE},
	"rshared": {
		invert:    false,
		recursive: true,
		flag:      unix.MS_SHARED | unix.MS_BIND,
	},
	"rslave":      {invert: false, recursive: true, flag: unix.MS_SLAVE},
	"runbindable": {invert: false, recursive: true, flag: unix.MS_UNBINDABLE},
	"rw":          {invert: true, recursive: false, flag: unix.MS_RDONLY},
	"shared":      {invert: false, recursive: false, flag: unix.MS_SHARED},
	"silent":      {invert: false, recursive: false, flag: unix.MS_SILENT},
	"slave":       {invert: false, recursive: false, flag: unix.MS_SLAVE},
	"strictatime": {
		invert:    false,
		recursive: true,
		flag:      unix.MS_STRICTATIME,
	},
	"suid": {invert: true, recursive: false, flag: unix.MS_NOSUID},
	"sync": {
		invert:    false,
		recursive: false,
		flag:      unix.MS_SYNCHRONOUS,
	},
	"unbindable": {
		invert:    false,
		recursive: false,
		flag:      unix.MS_UNBINDABLE,
	},
}

// MountRootfs mounts the container's root filesystem at given containerRootfs.
//
// NOTE: The actual mount operations are performed by the C constructor
// (nssetup.c) BEFORE Go starts. This ensures mount operations happen in a
// single-threaded context, which is critical for mount propagation to work
// correctly (especially for rshared).
//
// The C code follows runc's prepareRoot sequence:
// 1. Set "/" propagation based on rootfsPropagation (default: rslave)
// 2. Make rootfs's parent mount private (isolates rootfs from "/" propagation)
// 3. Bind mount rootfs to itself (makes it a proper mount point for pivot_root)
//
// This function is now a no-op but is kept for API compatibility.
func MountRootfs(containerRootfs string, rootfsPropagation string) error {
	// Mount operations are performed by the C constructor before Go starts.
	return nil
}

// rootfsParentMountPrivate makes the nearest parent mount point of path private.
// This prevents subsequent bind mounts from propagating to other namespaces.
func rootfsParentMountPrivate(path string) error {
	for {
		if err := unix.Mount("", path, "", unix.MS_PRIVATE, ""); err == nil {
			return nil
		} else if err != unix.EINVAL {
			return fmt.Errorf("remount-private %s: %w", path, err)
		}
		// EINVAL means not a mount point, try parent.
		if path == "/" {
			// Reached root - "/" is always a mount point so this shouldn't happen,
			// but if it does, just return nil as "/" being private is safe.
			return nil
		}
		path = filepath.Dir(path)
	}
}

// ParseRootfsPropagation converts a propagation string to mount flags.
func ParseRootfsPropagation(prop string) uintptr {
	switch prop {
	case "shared":
		return unix.MS_SHARED
	case "rshared":
		return unix.MS_SHARED | unix.MS_REC
	case "private":
		return unix.MS_PRIVATE
	case "rprivate":
		return unix.MS_PRIVATE | unix.MS_REC
	case "slave":
		return unix.MS_SLAVE
	case "rslave":
		return unix.MS_SLAVE | unix.MS_REC
	case "unbindable":
		return unix.MS_UNBINDABLE
	case "runbindable":
		return unix.MS_UNBINDABLE | unix.MS_REC
	default:
		return 0
	}
}

// MountRootReadonly remounts the root filesystem as read-only.
func MountRootReadonly() error {
	if err := Remount("/", unix.MS_BIND|unix.MS_RDONLY); err != nil {
		return fmt.Errorf("remount root as readonly: %w", err)
	}

	return nil
}

// SetRootfsMountPropagation sets the mount propagation for the root filesystem.
//
// Note: We don't use MS_REC here because submounts have their own propagation
// settings that should be preserved (e.g., rshared volumes).
func SetRootfsMountPropagation(prop string) error {
	var flag uintptr

	switch prop {
	case "shared", "rshared":
		flag = unix.MS_SHARED
	case "private", "rprivate":
		flag = unix.MS_PRIVATE
	case "slave", "rslave":
		flag = unix.MS_SLAVE
	case "unbindable", "runbindable":
		flag = unix.MS_UNBINDABLE
	case "":
		return nil // No propagation specified
	default:
		return nil // Unknown propagation, ignore
	}

	// Apply to root mount only (not recursively).
	if err := unix.Mount("", "/", "", flag, ""); err != nil {
		return fmt.Errorf("set mount propagation %s: %w", prop, err)
	}

	return nil
}
