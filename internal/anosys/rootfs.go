package anosys

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

type mountOption struct {
	flag      uintptr
	recursive bool
	invert    bool
}

// FIXME: commented out ones cause the runtime to hang for some reason; seems like they're shared subtrees
var mountOptions = map[string]mountOption{
	"async":         {invert: true, recursive: false, flag: unix.MS_SYNCHRONOUS},
	"atime":         {invert: true, recursive: true, flag: unix.MS_NOATIME},
	"bind":          {invert: false, recursive: false, flag: unix.MS_BIND},
	"defaults":      {invert: false, recursive: false, flag: 0},
	"dev":           {invert: true, recursive: false, flag: unix.MS_NODEV},
	"diratime":      {invert: true, recursive: false, flag: unix.MS_NODIRATIME},
	"dirsync":       {invert: false, recursive: false, flag: unix.MS_DIRSYNC},
	"exec":          {invert: true, recursive: false, flag: unix.MS_NOEXEC},
	"iversion":      {invert: false, recursive: false, flag: unix.MS_I_VERSION},
	"lazytime":      {invert: false, recursive: false, flag: unix.MS_LAZYTIME},
	"loud":          {invert: true, recursive: false, flag: unix.MS_SILENT},
	"noatime":       {invert: false, recursive: true, flag: unix.MS_NOATIME},
	"nodev":         {invert: false, recursive: true, flag: unix.MS_NODEV},
	"nodiratime":    {invert: false, recursive: true, flag: unix.MS_NODIRATIME},
	"noexec":        {invert: false, recursive: true, flag: unix.MS_NOEXEC},
	"noiversion":    {invert: true, recursive: false, flag: unix.MS_I_VERSION},
	"nolazytime":    {invert: true, recursive: false, flag: unix.MS_LAZYTIME},
	"norelatime":    {invert: true, recursive: false, flag: unix.MS_RELATIME},
	"nostrictatime": {invert: true, recursive: false, flag: unix.MS_STRICTATIME},
	"nosuid":        {invert: false, recursive: true, flag: unix.MS_NOSUID},
	"nosymfollow":   {invert: false, recursive: true, flag: unix.MS_NOSYMFOLLOW},
	// "private":       {invert: false, recursive: false, flag: unix.MS_PRIVATE},
	"rbind":    {invert: false, recursive: true, flag: unix.MS_BIND},
	"relatime": {invert: false, recursive: true, flag: unix.MS_RELATIME},
	"remount":  {invert: false, recursive: false, flag: unix.MS_REMOUNT},
	"ro":       {invert: false, recursive: true, flag: unix.MS_RDONLY},
	// "rprivate":      {invert: false, recursive: true, flag: unix.MS_PRIVATE},
	"rshared":     {invert: false, recursive: true, flag: unix.MS_SHARED | unix.MS_BIND},
	"rslave":      {invert: false, recursive: true, flag: unix.MS_SLAVE},
	"runbindable": {invert: false, recursive: true, flag: unix.MS_UNBINDABLE},
	"rw":          {invert: true, recursive: false, flag: unix.MS_RDONLY},
	// "shared":      {invert: false, recursive: false, flag: unix.MS_SHARED},
	"silent": {invert: false, recursive: false, flag: unix.MS_SILENT},
	// "slave":       {invert: false, recursive: false, flag: unix.MS_SLAVE},
	"strictatime": {invert: false, recursive: true, flag: unix.MS_STRICTATIME},
	"suid":        {invert: true, recursive: false, flag: unix.MS_NOSUID},
	"sync":        {invert: false, recursive: false, flag: unix.MS_SYNCHRONOUS},
	"unbindable":  {invert: false, recursive: false, flag: unix.MS_UNBINDABLE},
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
		f.flag,
		"",
	); err != nil {
		return fmt.Errorf("set rootfs mount propagation (%s): %w", prop, err)
	}

	return nil
}
