package anosys

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func MountSpecMounts(mounts []specs.Mount, rootfs string) error {
	for _, m := range mounts {
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
			} else if opt == "newinstance" {
				dataOptions = append(dataOptions, "newinstance")
			} else {
				logrus.Warn("mount option not captured: ", opt)
			}
		}

		// FIXME: don't know why tmpfs with shared, slave or private doesn't work;
		//				gives a 'no such file or directory' error when trying to mount
		//				skip these until we figure it out
		if m.Source == "tmpfs" && m.Type == "tmpfs" &&
			(slices.Contains(m.Options, "shared") || slices.Contains(m.Options, "slave") || slices.Contains(m.Options, "private")) {
			continue
		}

		if err := syscall.Mount(
			m.Source,
			dest,
			m.Type,
			uintptr(flags),
			strings.Join(dataOptions, ","),
		); err != nil {
			logrus.Error("mount source: ", m.Source)
			logrus.Error("mount dest: ", dest)
			logrus.Error("mount type: ", m.Type)
			logrus.Error("mount data: ", dataOptions)
			return fmt.Errorf("mount spec mount (%s): %w", dest, err)
		}
	}

	return nil
}
