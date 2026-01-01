package platform

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// ErrInvalidNamespacePath is returned when an invalid namespace path is
// specified.
var ErrInvalidNamespacePath = errors.New("invalid namespace path")

// NamespaceFlags maps LinuxNamespaceType to corresponding Linux clone flags.
var NamespaceFlags = map[specs.LinuxNamespaceType]uintptr{
	specs.PIDNamespace:     unix.CLONE_NEWPID,
	specs.NetworkNamespace: unix.CLONE_NEWNET,
	specs.MountNamespace:   unix.CLONE_NEWNS,
	specs.IPCNamespace:     unix.CLONE_NEWIPC,
	specs.UTSNamespace:     unix.CLONE_NEWUTS,
	specs.UserNamespace:    unix.CLONE_NEWUSER,
	specs.CgroupNamespace:  unix.CLONE_NEWCGROUP,
	specs.TimeNamespace:    unix.CLONE_NEWTIME,
}

// NamespaceEnvs maps LinuxNamespaceType to corresponding environment flag
// names.
var NamespaceEnvs = map[specs.LinuxNamespaceType]string{
	specs.PIDNamespace:     "pid",
	specs.NetworkNamespace: "net",
	specs.MountNamespace:   "mnt",
	specs.IPCNamespace:     "ipc",
	specs.UTSNamespace:     "uts",
	specs.UserNamespace:    "user",
	specs.CgroupNamespace:  "cgroup",
	specs.TimeNamespace:    "time",
}

// SetNS enters the namespace specified by the given path.
func SetNS(path string) error {
	fd, err := unix.Open(path, unix.O_RDONLY, 0o666)
	if err != nil {
		return fmt.Errorf("open ns path %s: %w", path, err)
	}
	defer unix.Close(fd)

	_, _, errno := unix.Syscall(unix.SYS_SETNS, uintptr(fd), 0, 0)
	if errno != 0 {
		return fmt.Errorf("set namespace %s errno: %w", path, errno)
	}

	return nil
}

// ValidateNSPath validates that the suffix of the Path is valid for the Type
// in the given ns.
func ValidateNSPath(ns *specs.LinuxNamespace) error {
	if ns.Path == "" {
		return nil
	}

	f, err := os.Open(ns.Path)
	if err != nil {
		return fmt.Errorf("open namespace path: %w", err)
	}

	nsType, err := unix.IoctlRetInt(int(f.Fd()), unix.NS_GET_NSTYPE)
	if err != nil {
		return fmt.Errorf("get namespace type from file descriptor: %w", err)
	}

	if NamespaceFlags[ns.Type] != uintptr(nsType) {
		return ErrInvalidNamespacePath
	}

	return nil
}

// BuildUserNSMappings converts UID/GID mappings from an OCI spec to
// syscall.SysProcIDMap for user namespace configuration via cmd.Exec.
// If no mappings are provided then it defaults to mapping the current process'
// UID/GID.
func BuildUserNSMappings(
	specUIDMappings []specs.LinuxIDMapping,
	specGIDMappings []specs.LinuxIDMapping,
) ([]syscall.SysProcIDMap, []syscall.SysProcIDMap) {
	uidMappings := make([]syscall.SysProcIDMap, 0, 1)
	gidMappings := make([]syscall.SysProcIDMap, 0, 1)

	if uidCount := len(specUIDMappings); uidCount > 0 {
		uidMappings = slices.Grow(uidMappings, uidCount-1)

		for _, m := range specUIDMappings {
			uidMappings = append(uidMappings, syscall.SysProcIDMap{
				ContainerID: int(m.ContainerID),
				HostID:      int(m.HostID),
				Size:        int(m.Size),
			})
		}
	} else {
		uidMappings = append(uidMappings, syscall.SysProcIDMap{
			ContainerID: 0,
			HostID:      os.Getuid(),
			Size:        1,
		})
	}

	if gidCount := len(specGIDMappings); gidCount > 0 {
		gidMappings = slices.Grow(gidMappings, gidCount-1)

		for _, m := range specGIDMappings {
			gidMappings = append(gidMappings, syscall.SysProcIDMap{
				ContainerID: int(m.ContainerID),
				HostID:      int(m.HostID),
				Size:        int(m.Size),
			})
		}
	} else {
		gidMappings = append(gidMappings, syscall.SysProcIDMap{
			ContainerID: 0,
			HostID:      os.Getgid(),
			Size:        1,
		})
	}

	return uidMappings, gidMappings
}
