package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// MountSpecMounts mounts the specified mounts into the containers root
// filesystem.
func MountSpecMounts(mounts []specs.Mount, rootfs string) error {
	for _, m := range mounts {
		var flags uintptr

		// For cgroupv2 bind mount the cgroup hierarchy.
		if m.Type == "cgroup" && IsUnifiedCgroupsMode() {
			if err := BindMount(
				"/sys/fs/cgroup",
				filepath.Join(rootfs, m.Destination),
				true,
			); err != nil {
				return fmt.Errorf("bind mount cgroup2: %w", err)
			}

			continue
		}

		dest := filepath.Join(rootfs, m.Destination)

		if _, err := os.Stat(dest); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("exists (%s): %w", dest, err)
			}

			if m.Type == "bind" || slices.Contains(m.Options, "bind") ||
				slices.Contains(m.Options, "rbind") {

				srcInfo, err := os.Stat(m.Source)
				if err != nil {
					return fmt.Errorf("stat mount source: %w", err)
				}

				if srcInfo.IsDir() {
					if err := os.MkdirAll(dest, os.ModeDir); err != nil {
						return fmt.Errorf(
							"make mount dir target (%s): %w",
							dest,
							err,
						)
					}
				} else {
					if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
						return fmt.Errorf("make mount parent dir (%s): %w", filepath.Dir(dest), err)
					}

					f, err := os.Create(dest)
					if err != nil {
						return fmt.Errorf("make mount target file (%s): %w", dest, err)
					}
					f.Close()
				}
			} else {
				if err := os.MkdirAll(dest, 0o755); err != nil {
					return fmt.Errorf(
						"make mount dir (%s): %w",
						filepath.Dir(dest),
						err,
					)
				}
			}
		}

		var dataOptions []string

		for _, opt := range m.Options {
			if f, ok := mountOptions[opt]; ok {
				if propagateBindMount(opt) {
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

		if err := MountFilesystem(
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

func propagateBindMount(opt string) bool {
	return opt != "private" && opt != "rprivate" &&
		opt != "shared" &&
		opt != "rshared" &&
		opt != "slave" &&
		opt != "rslave"
}
