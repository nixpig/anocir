package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// MountSpecMounts mounts the specified mounts into the container's root
// filesystem.
func MountSpecMounts(mounts []specs.Mount, rootfs string) error {
	for _, m := range mounts {
		logrus.Debug("mounting: ", m)

		var flags uintptr

		/*
			TODO: in Docker trying to mount cgroup mountpoint if cgroupv2 is enabled doesn't work
						the call to `mount` results in an 'invalid argument' error
						need to find out if that's the expected behaviour or not
		*/
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

		var dataOptions []string

		for _, opt := range m.Options {
			if f, ok := mountOptions[opt]; ok {
				// bind mount propagation
				propagate := opt != "private" && opt != "rprivate" &&
					opt != "shared" &&
					opt != "rshared" &&
					opt != "slave" &&
					opt != "rslave"

				if propagate {
					if f.invert {
						flags &= f.flag
					} else {
						flags |= f.flag
					}
				}

				if f.recursive {
					flags |= unix.MS_REC
				}
			} else if strings.Contains(opt, "=") {
				dataOptions = append(dataOptions, opt)
			}
		}

		logrus.Debug("data: ", dataOptions)

		if err := syscall.Mount(
			m.Source,
			dest,
			m.Type,
			uintptr(flags),
			strings.Join(dataOptions, ","),
		); err != nil {
			return fmt.Errorf("mount spec mount: %w", err)
		}
	}

	return nil
}
