package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func mountRootfs(containerRootfs string) error {
	dev1 := Device{
		Source: "",
		Target: "/",
		Fstype: "",
		Flags:  unix.MS_PRIVATE | unix.MS_REC,
		Data:   "",
	}
	if err := dev1.Mount(); err != nil {
		return err
	}

	dev2 := Device{
		Source: containerRootfs,
		Target: containerRootfs,
		Fstype: "",
		Flags:  unix.MS_BIND | unix.MS_REC,
		Data:   "",
	}

	if err := dev2.Mount(); err != nil {
		return err
	}

	return nil
}

func mountProc(containerRootfs string) error {
	containerProc := filepath.Join(containerRootfs, "proc")
	if err := os.MkdirAll(containerProc, 0666); err != nil {
		return fmt.Errorf("create proc dir: %w", err)
	}

	dev := Device{
		Source: "proc",
		Target: containerProc,
		Fstype: "proc",
		Flags:  uintptr(0),
		Data:   "",
	}

	if err := dev.Mount(); err != nil {
		return err
	}

	return nil
}

func devIsInSpec(mounts []specs.Mount, dev string) bool {
	return slices.ContainsFunc(mounts, func(m specs.Mount) bool {
		return m.Destination == dev
	})
}

func mountDevices(devices []specs.LinuxDevice, rootfs string) error {
	for _, d := range devices {
		absPath := filepath.Join(rootfs, strings.TrimPrefix(d.Path, "/"))

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			f, err := os.Create(absPath)
			if err != nil && !os.IsExist(err) {
				return err
			}
			if f != nil {
				f.Close()
			}
		}

		dev := Device{
			Source: d.Path,
			Target: absPath,
			Fstype: "bind",
			Flags:  unix.MS_BIND,
			Data:   "",
		}

		if err := dev.Mount(); err != nil {
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
			flags |= unix.MS_BIND
		}

		var dataOptions []string
		for _, opt := range mount.Options {
			if opt == "bind" || opt == "rbind" {
				mount.Type = "bind"
				flags |= unix.MS_BIND
			}
		}

		var data string
		if len(dataOptions) > 0 {
			data = strings.Join(dataOptions, ",")
		}

		dev := Device{
			Source: mount.Source,
			Target: dest,
			Fstype: mount.Type,
			Flags:  uintptr(flags),
			Data:   data,
		}

		if err := dev.Mount(); err != nil {
			return fmt.Errorf("mount device (%+v): %w", dev, err)
		}
	}

	return nil
}
