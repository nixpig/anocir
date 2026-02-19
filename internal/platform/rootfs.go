package platform

import (
	"fmt"

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
// Sets "/" to rslave recursively to allow mounts from the host to propagate
// into the container (for rslave mounts) while preventing container mounts
// from affecting the host. Individual mount propagation settings (rshared,
// rslave, etc...) need to be applied to specific mounts separately.
func MountRootfs(containerRootfs string) error {
	if err := SetPropagation("/", unix.MS_SLAVE|unix.MS_REC); err != nil {
		return err
	}

	if err := BindMount(containerRootfs, containerRootfs, true); err != nil {
		return fmt.Errorf("bind mount rootfs: %w", err)
	}

	return nil
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
	flag := getPropagationFlag(prop) &^ unix.MS_REC

	if flag == 0 {
		return nil
	}

	// Apply to root mount only (not recursively).
	if err := unix.Mount("", "/", "", flag, ""); err != nil {
		return fmt.Errorf("set mount propagation %s: %w", prop, err)
	}

	return nil
}
