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
		// TODO: docker run doesn't work unless we skip; figure out _why_ and if it's the correct behaviour
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

		for _, opt := range m.Options {
			if f, ok := mountOptions[opt]; ok {
				if f.invert {
					flags &= f.flag
				} else {
					flags |= f.flag
				}

				if f.recursive {
					flags |= unix.MS_REC
				}
			}
		}

		// FIXME: what is options supposed to contain??
		var dataOptions []string
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
