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
		// TODO: this doesn't seem right to me, but docker run doesn't work unless we skip; figure out _why_
		if m.Type == "cgroup" && IsUnifiedCGroupsMode() {
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
		switch m.Type {
		case "bind":
			flags |= unix.MS_BIND
		case "rbind":
			flags |= unix.MS_BIND | unix.MS_REC
		}

		var dataOptions []string
		for _, opt := range m.Options {
			switch opt {
			case "bind":
				flags |= unix.MS_BIND
			case "rbind":
				flags |= unix.MS_BIND | unix.MS_REC
			}
		}

		// TODO: this should be from the m.Options but it freezes when set :/
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
			return fmt.Errorf("mount spec mount: %w", err)
		}
	}

	return nil
}
