package anosys

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func MountSpecMounts(mounts []specs.Mount, rootfs string) error {
	for _, m := range mounts {
		logrus.Info("---")
		logrus.Info("mounting: ", m)

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
				return fmt.Errorf("stat mount destination (%s): %w", dest, err)
			}

			if err := os.MkdirAll(dest, os.ModeDir); err != nil {
				return fmt.Errorf("create mount destination dir (%s): %w", dest, err)
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
				if opt == "private" || opt == "shared" || opt == "slave" {
					flags |= unix.MS_BIND
				} else if opt == "rprivate" || opt == "rshared" || opt == "rslave" {
					flags |= unix.MS_BIND | unix.MS_REC
				}

				if f.invert {
					flags &= ^f.flag
				} else {
					flags |= f.flag
				}

				if f.recursive {
					flags |= unix.MS_REC
				}
			} else if strings.Contains(opt, "=") {
				dataOptions = append(dataOptions, opt)

				optParts := strings.Split(opt, "=")
				if len(optParts) > 1 && optParts[0] == "mode" {
					mode, err := strconv.ParseUint(optParts[1], 8, 32)
					if err != nil {
						return fmt.Errorf("parse mount destination mode from data opts: %w", err)
					}

					if err := os.Chmod(dest, os.FileMode(mode)); err != nil {
						return fmt.Errorf("set mount destination mode from data opts: %w", err)
					}
				}
			}
		}

		logrus.Info("source: ", m.Source)
		logrus.Info("dest: ", dest)
		logrus.Info("type: ", m.Type)
		logrus.Info("data: ", dataOptions)
		logrus.Info("---")

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
