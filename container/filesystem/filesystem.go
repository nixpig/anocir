package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func mountDevice(device Device, log *zerolog.Logger) error {
	if _, err := os.Stat(device.Target); os.IsNotExist(err) {
		log.Info().Str("target", device.Target).Msg("create target")
		f, err := os.Create(device.Target)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("create device target if not exists: %w", err)
		}
		if f != nil {
			f.Close()
		}
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

func mountRootfs(containerRootfs string, log *zerolog.Logger) error {
	if err := mountDevice(Device{
		Source: "",
		Target: "/",
		Fstype: "",
		Flags:  syscall.MS_PRIVATE | syscall.MS_REC,
		Data:   "",
	}, log); err != nil {
		return err
	}

	if err := mountDevice(Device{
		Source: containerRootfs,
		Target: containerRootfs,
		Fstype: "",
		Flags:  syscall.MS_BIND | syscall.MS_REC,
		Data:   "",
	}, log); err != nil {
		return err
	}

	return nil
}

func mountProc(containerRootfs string, log *zerolog.Logger) error {
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
	}, log); err != nil {
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

func mountDevices(devices []specs.LinuxDevice, rootfs string, log *zerolog.Logger) error {
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
		}

		if err := mountDevice(Device{
			Source: dev.Path,
			Target: absPath,
			Fstype: "bind",
			Flags:  syscall.MS_BIND,
			Data:   "",
		}, log); err != nil {
			return fmt.Errorf("mount device: %w", err)
		}
	}

	return nil
}

func mountSpecMounts(mounts []specs.Mount, rootfs string, log *zerolog.Logger) error {
	for _, mount := range mounts {
		dest := filepath.Join(rootfs, mount.Destination)

		if _, err := os.Stat(dest); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("exists: %w", err)
			}

			if err := os.MkdirAll(dest, os.ModeDir); err != nil {
				return fmt.Errorf("make dir: %w", err)
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
			// o, ok := mountOptions[opt]
			// if !ok {
			// 	if !strings.HasPrefix(opt, "gid=") &&
			// 		!strings.HasPrefix(opt, "uid=") &&
			// 		opt != "newinstance" {
			// 		dataOptions = append(dataOptions, opt)
			// 	}
			// } else {
			// 	log.Info().Str("opt", opt).Any("flag", o.Flag).Msg("LISTED OPTION")
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

		if err := mountDevice(Device{
			Source: mount.Source,
			Target: dest,
			Fstype: mount.Type,
			Flags:  uintptr(flags),
			Data:   data,
		}, log); err != nil {
			log.Error().Str("source", mount.Source).Str("target", dest).Msg("NO SUCH DEVICE???")
			return fmt.Errorf("mount device: %w", err)
		}
	}

	return nil
}
