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

// MountSpecMounts mounts the given mounts into the given containers
// containerRootfs.
func MountSpecMounts(mounts []specs.Mount, containerRootfs string) error {
	for _, m := range mounts {
		var flags uintptr

		// For cgroupv2 bind mount the cgroup hierarchy.
		if m.Type == "cgroup" && IsUnifiedCgroupsMode() {
			if err := BindMount(
				"/sys/fs/cgroup",
				filepath.Join(containerRootfs, m.Destination),
				true,
			); err != nil {
				return fmt.Errorf("bind mount cgroup2: %w", err)
			}

			continue
		}

		dest := filepath.Join(containerRootfs, m.Destination)

		if _, err := os.Stat(dest); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("exists (%s): %w", dest, err)
			}

			if m.Type == "bind" || slices.Contains(m.Options, "bind") ||
				slices.Contains(m.Options, "rbind") {

				if info, err := os.Lstat(dest); err == nil && info.Mode()&os.ModeSymlink != 0 {
					if err := os.Remove(dest); err != nil {
						return fmt.Errorf("remove symlink before mount (%s): %w", dest, err)
					}

					f, err := os.Create(dest)
					if err != nil {
						return fmt.Errorf("recreate symlink as regular file for mount (%s): %w", dest, err)
					}
					f.Close()
				}

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
		var propagationFlag uintptr
		var recursiveReadonly bool

		for _, opt := range m.Options {
			// Handle propagation options separately.
			if pf := getPropagationFlag(opt); pf != 0 {
				propagationFlag = pf
				continue
			}

			// Handle recursive readonly (rro) separately - requires mount_setattr.
			if opt == "rro" {
				recursiveReadonly = true
				flags |= unix.MS_RDONLY // Set regular readonly.
				continue
			}

			if f, ok := mountOptions[opt]; ok {
				if f.invert {
					flags &^= f.flag
				} else {
					flags |= f.flag
				}
				if f.recursive {
					flags |= unix.MS_REC
				}
			} else if strings.Contains(opt, "=") {
				dataOptions = append(dataOptions, opt)
			}
		}

		// For rshared mounts, we need to ensure the source is shared BEFORE
		// the bind mount so the new mount joins the source's peer group.
		// This enables bidirectional propagation between host and container.
		isRshared := propagationFlag == (unix.MS_SHARED | unix.MS_REC)
		if isRshared && (m.Type == "bind" || slices.Contains(m.Options, "bind") ||
			slices.Contains(m.Options, "rbind")) {
			// Make the source shared before bind mounting.
			// This ensures the bind mount joins the source's peer group.
			if err := SetPropagation(m.Source, unix.MS_SHARED|unix.MS_REC); err != nil {
				// If setting source propagation fails, continue anyway.
				// The mount will still work, just without bidirectional propagation.
				fmt.Fprintf(os.Stderr, "warning: failed to set source propagation for rshared mount: %v\n", err)
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

		// Apply propagation after the initial mount.
		// For rshared, this reinforces the shared propagation.
		if propagationFlag != 0 {
			if err := SetPropagation(dest, propagationFlag); err != nil {
				return fmt.Errorf("set mount propagation: %w", err)
			}
		}

		// Apply recursive readonly using mount_setattr.
		if recursiveReadonly {
			if err := setRecursiveReadonly(dest); err != nil {
				return fmt.Errorf("set recursive readonly: %w", err)
			}
		}
	}

	return nil
}

// getPropagationFlag returns the mount propagation flag for the given opt.
// Returns 0 if not a propagation option.
func getPropagationFlag(opt string) uintptr {
	switch opt {
	case "private":
		return unix.MS_PRIVATE
	case "rprivate":
		return unix.MS_PRIVATE | unix.MS_REC
	case "shared":
		return unix.MS_SHARED
	case "rshared":
		return unix.MS_SHARED | unix.MS_REC
	case "slave":
		return unix.MS_SLAVE
	case "rslave":
		return unix.MS_SLAVE | unix.MS_REC
	case "unbindable":
		return unix.MS_UNBINDABLE
	case "runbindable":
		return unix.MS_UNBINDABLE | unix.MS_REC
	default:
		return 0
	}
}

// setRecursiveReadonly makes a mount and all its submounts read-only
// using the mount_setattr syscall with AT_RECURSIVE.
func setRecursiveReadonly(path string) error {
	attr := unix.MountAttr{
		Attr_set: unix.MOUNT_ATTR_RDONLY,
	}

	return unix.MountSetattr(-1, path, unix.AT_RECURSIVE, &attr)
}
