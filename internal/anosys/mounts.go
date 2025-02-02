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
	for _, m := range mounts {
		// added to satisfy 'docker run' issue
		// TODO: figure out _why_
		if m.Type == "cgroup" {
			continue
		}

		dest := filepath.Join(rootfs, m.Destination)

		if _, err := os.Stat(dest); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("exists (%s): %w", dest, err)
			}

			if err := os.MkdirAll(dest, os.ModeDir); err != nil {
				return fmt.Errorf("make dir (%s): %w", dest, err)
			}
		}

		var flags uintptr
		if m.Type == "bind" {
			flags |= unix.MS_BIND
		}

		var dataOptions []string
		for _, opt := range m.Options {
			if opt == "bind" || opt == "rbind" {
				m.Type = "bind"
				flags |= unix.MS_BIND
			}
		}

		var data string
		if len(dataOptions) > 0 {
			data = strings.Join(dataOptions, ",")
		}

		if err := syscall.Mount(
			m.Source,
			dest,
			m.Type,
			uintptr(flags),
			data,
		); err != nil {
			return fmt.Errorf(
				"mount spec mount (%s, %s, %s, %s, %s): %w",
				m.Source,
				dest,
				m.Type,
				flags,
				data,
				err,
			)
		}
	}

	return nil
}
