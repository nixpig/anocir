package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func MountProc(containerRootfs string) error {
	containerPath := filepath.Join(containerRootfs, "proc")

	if err := os.MkdirAll(
		containerPath,
		os.ModeDir,
	); err != nil {
		return err
	}

	if err := syscall.Mount(
		"proc",
		containerPath,
		"proc",
		uintptr(0),
		"",
	); err != nil {
		return err
	}

	return nil
}

func MountRootfs(containerRootfs string) error {
	if err := syscall.Mount(
		"",
		"/",
		"",
		syscall.MS_PRIVATE|syscall.MS_REC,
		"",
	); err != nil {
		return err
	}

	if err := syscall.Mount(
		containerRootfs,
		containerRootfs,
		"",
		syscall.MS_BIND|syscall.MS_REC,
		"",
	); err != nil {
		return err
	}

	return nil
}

func DevInSpec(mounts []specs.Mount, dev string) bool {
	for _, mount := range mounts {
		if mount.Destination == dev {
			return true
		}
	}

	return false
}

func MountDevices(devices []specs.LinuxDevice, rootfs string) error {
	for _, dev := range devices {
		var absPath string
		if strings.Index(dev.Path, "/") == 0 {
			relPath := strings.TrimPrefix(dev.Path, "/")
			absPath = filepath.Join(rootfs, relPath)
		} else {
			absPath = filepath.Join(rootfs, dev.Path)
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			f, err := os.Create(absPath)
			if err != nil && !os.IsExist(err) {
				return err
			}
			if f != nil {
				f.Close()
			}
			if err := syscall.Mount(
				dev.Path,
				absPath,
				"bind",
				syscall.MS_BIND,
				"",
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func CreateSymlinks(symlinks map[string]string, rootfs string) error {
	for src, dest := range symlinks {
		if err := os.Symlink(
			src,
			filepath.Join(rootfs, dest),
		); err != nil {
			return err
		}
	}

	return nil
}

func MountSpecMounts(mounts []specs.Mount, rootfs string) error {
	for _, mount := range mounts {
		var dest string
		if strings.Index(mount.Destination, "/") == 0 {
			dest = filepath.Join(rootfs, mount.Destination)
		} else {
			dest = mount.Destination
		}

		if _, err := os.Stat(dest); err != nil {
			if !os.IsNotExist(err) {
				return err
			}

			if err := os.MkdirAll(dest, os.ModeDir); err != nil {
				return err
			}
		}

		var flags uintptr
		if mount.Type == "bind" {
			flags |= syscall.MS_BIND
		}

		var dataOptions []string
		for _, opt := range mount.Options {
			o, ok := MountOptions[opt]
			if !ok {
				if !strings.HasPrefix(opt, "gid=") &&
					!strings.HasPrefix(opt, "uid=") &&
					opt != "newinstance" {
					dataOptions = append(dataOptions, opt)
				}
			} else {
				if !o.No {
					flags |= o.Flag
				}
			}
		}

		var data string
		if len(dataOptions) > 0 {
			data = strings.Join(dataOptions, ",")
		}

		if err := syscall.Mount(
			mount.Source,
			dest,
			mount.Type,
			flags,
			data,
		); err != nil {
			return err
		}
	}

	return nil
}
