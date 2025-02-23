package anosys

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

func MountSpecMounts(mounts []specs.Mount, rootfs string) error {
	f, _ := os.Create("/var/log/anocir/log.txt")
	logrus.SetOutput(f)

	for _, m := range mounts {
		logrus.Info("----------------------------------")
		logrus.Info("mounting: ", m)

		var flags uintptr

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

		var dataOptions []string

		for _, opt := range m.Options {
			// TODO: should this be setting the group id of the mount?
			switch o := opt; {
			case strings.HasPrefix(o, "gid"):
				continue
			}

			if f, ok := mountOptions[opt]; ok {
				// bind mount propagation
				if opt != "private" && opt != "rprivate" && opt != "shared" && opt != "rshared" && opt != "slave" && opt != "rslave" {
					if f.invert {
						flags &= f.flag
					} else {
						flags |= f.flag
					}
				}

				if f.recursive {
					flags |= unix.MS_REC
				}
			} else if strings.Index(opt, "=") != -1 {
				dataOptions = append(dataOptions, opt)
			}
		}

		logrus.Info("data: ", dataOptions)

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
