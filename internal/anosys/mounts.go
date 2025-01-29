package anosys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func MountSpecMounts(mounts []specs.Mount, rootfs string) error {
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

		if err := syscall.Mount(
			mount.Source,
			dest,
			mount.Type,
			uintptr(flags),
			data,
		); err != nil {
			return fmt.Errorf("mount spec mount: %w", err)
		}
	}

	return nil
}
