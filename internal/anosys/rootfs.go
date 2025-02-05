package anosys

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

var mountOptions = map[string]uintptr{
	"async":         unix.MS_SYNCHRONOUS,
	"atime":         unix.MS_NOATIME,
	"bind":          unix.MS_BIND,
	"defaults":      0,
	"dev":           unix.MS_NODEV,
	"diratime":      unix.MS_NODIRATIME,
	"dirsync":       unix.MS_DIRSYNC,
	"exec":          unix.MS_NOEXEC,
	"iversion":      unix.MS_I_VERSION,
	"lazytime":      unix.MS_LAZYTIME,
	"loud":          unix.MS_SILENT,
	"noatime":       unix.MS_NOATIME,
	"nodev":         unix.MS_NODEV,
	"nodiratime":    unix.MS_NODIRATIME,
	"noexec":        unix.MS_NOEXEC,
	"noiversion":    unix.MS_I_VERSION,
	"nolazytime":    unix.MS_LAZYTIME,
	"norelatime":    unix.MS_RELATIME,
	"nostrictatime": unix.MS_STRICTATIME,
	"nosuid":        unix.MS_NOSUID,
	"nosymfollow":   unix.MS_NOSYMFOLLOW,
	"private":       unix.MS_PRIVATE,
	"rbind":         unix.MS_BIND | unix.MS_REC,
	"relatime":      unix.MS_RELATIME,
	"remount":       unix.MS_REMOUNT,
	"ro":            unix.MS_RDONLY,
	"rprivate":      unix.MS_PRIVATE | unix.MS_REC,
	"rshared":       unix.MS_SHARED | unix.MS_REC,
	"rslave":        unix.MS_SLAVE | unix.MS_REC,
	"runbindable":   unix.MS_UNBINDABLE | unix.MS_REC,
	"rw":            unix.MS_RDONLY,
	"shared":        unix.MS_SHARED,
	"silent":        unix.MS_SILENT,
	"slave":         unix.MS_SLAVE,
	"strictatime":   unix.MS_STRICTATIME,
	"suid":          unix.MS_NOSUID,
	"sync":          unix.MS_SYNCHRONOUS,
	"unbindable":    unix.MS_UNBINDABLE,
}

func MountRootfs(containerRootfs string) error {
	if err := syscall.Mount(
		"",
		"/",
		"",
		unix.MS_PRIVATE|unix.MS_REC,
		"",
	); err != nil {
		return err
	}

	if err := syscall.Mount(
		containerRootfs,
		containerRootfs,
		"",
		unix.MS_BIND|unix.MS_REC,
		"",
	); err != nil {
		return err
	}

	return nil
}

func MountRootReadonly() error {
	if err := syscall.Mount(
		"",
		"/",
		"",
		unix.MS_BIND|unix.MS_REMOUNT|unix.MS_RDONLY,
		"",
	); err != nil {
		return fmt.Errorf("remount root as readonly: %w", err)
	}

	return nil
}

func SetRootfsMountPropagation(prop string) error {
	f, ok := mountOptions[prop]
	if !ok {
		return nil
	}

	if err := syscall.Mount(
		"",
		"/",
		"",
		f,
		"",
	); err != nil {
		return fmt.Errorf("set rootfs mount propagation (%s): %w", prop, err)
	}

	return nil
}
