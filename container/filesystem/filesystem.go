package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func mountDevice(device Device) error {
	if _, err := os.Stat(device.Target); os.IsNotExist(err) {
		f, err := os.Create(device.Target)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("create device target if not exists: %w", err)
		}
		if f != nil {
			f.Close()
		}
	}

	if device.Fstype == "cgroup" {
		return nil
	}

	if err := syscall.Mount(
		device.Source,
		device.Target,
		device.Fstype,
		device.Flags,
		device.Data,
	); err != nil {
		return fmt.Errorf("mounting device: %w", err)
	}

	return nil
}

func mountRootfs(containerRootfs string) error {
	if err := mountDevice(Device{
		Source: "",
		Target: "/",
		Fstype: "",
		Flags:  syscall.MS_PRIVATE | syscall.MS_REC,
		Data:   "",
	}); err != nil {
		return err
	}

	if err := mountDevice(Device{
		Source: containerRootfs,
		Target: containerRootfs,
		Fstype: "",
		Flags:  syscall.MS_BIND | syscall.MS_REC,
		Data:   "",
	}); err != nil {
		return err
	}

	return nil
}

func mountProc(containerRootfs string) error {
	containerProc := filepath.Join(containerRootfs, "proc")
	if err := os.MkdirAll(containerProc, 0666); err != nil {
		return fmt.Errorf("create proc dir: %w", err)
	}

	if err := mountDevice(Device{
		Source: "proc",
		Target: containerProc,
		Fstype: "proc",
		Flags:  uintptr(0),
		Data:   "",
	}); err != nil {
		return err
	}

	return nil
}

func devIsInSpec(mounts []specs.Mount, dev string) bool {
	for _, mount := range mounts {
		if mount.Destination == dev {
			return true
		}
	}

	return false
}

func mountDevices(devices []specs.LinuxDevice, rootfs string) error {
	for _, dev := range devices {
		absPath := filepath.Join(rootfs, strings.TrimPrefix(dev.Path, "/"))

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			f, err := os.Create(absPath)
			if err != nil && !os.IsExist(err) {
				return err
			}
			if f != nil {
				f.Close()
			}
		}

		if err := mountDevice(Device{
			Source: dev.Path,
			Target: absPath,
			Fstype: "bind",
			Flags:  syscall.MS_BIND,
			Data:   "",
		}); err != nil {
			return fmt.Errorf("mount device: %w", err)
		}
	}

	return nil
}

func mountSpecMounts(mounts []specs.Mount, rootfs string) error {
	for _, mount := range mounts {
		dest := filepath.Join(rootfs, mount.Destination)

		if _, err := os.Stat(dest); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("exists (%s): %w", dest, err)
			}

			if err := os.MkdirAll(dest, os.ModeDir); err != nil {
				return fmt.Errorf("make dir (%s): %w", dest, err)
			}
		}

		var flags uintptr
		if mount.Type == "bind" {
			flags |= syscall.MS_BIND
		}

		var dataOptions []string
		for _, opt := range mount.Options {
			if opt == "bind" || opt == "rbind" {
				mount.Type = "bind"
				flags |= syscall.MS_BIND
			}

			// TODO: review why this breaks everything!!
			// o, ok := MountOptions[opt]
			// if !ok {
			// 	if !strings.HasPrefix(opt, "gid=") &&
			// 		!strings.HasPrefix(opt, "uid=") &&
			// 		opt != "newinstance" {
			// 		dataOptions = append(dataOptions, opt)
			// 	}
			// } else {
			// 	if !o.No {
			// 		flags |= o.Flag
			// 	} else {
			// 		flags ^= o.Flag
			// 	}
			// }
		}

		var data string
		if len(dataOptions) > 0 {
			data = strings.Join(dataOptions, ",")
		}

		d := Device{
			Source: mount.Source,
			Target: dest,
			Fstype: mount.Type,
			Flags:  uintptr(flags),
			Data:   data,
		}

		if err := mountDevice(d); err != nil {
			return fmt.Errorf("mount device (%+v): %w", d, err)
		}
	}

	return nil
}
