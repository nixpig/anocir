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
func SetNS(fd uintptr) error {
	_, _, errno := unix.Syscall(unix.SYS_SETNS, uintptr(fd), 0, 0)
	if errno != 0 {
		return fmt.Errorf("set namespace errno: %w", errno)
	}

	return nil
}

// OpenNSPath opens the path of the given ns and validates it, returning the
// corresponding file. If validation fails then ErrInvalidNamespacePath error
// is returned.
func OpenNSPath(ns *specs.LinuxNamespace) (*os.File, error) {
	if ns.Path == "" {
		return nil, ErrInvalidNamespacePath
	}

	f, err := os.Open(ns.Path)
	if err != nil {
		return nil, fmt.Errorf("open namespace path: %w", err)
	}

	nsType, err := unix.IoctlRetInt(int(f.Fd()), unix.NS_GET_NSTYPE)
	if err != nil {
		return nil, fmt.Errorf(
			"get namespace type from file descriptor: %w",
			err,
		)
	}

	if NamespaceFlags[ns.Type] != uintptr(nsType) {
		return nil, ErrInvalidNamespacePath
	}

	return f, nil
}

// BuildUserNSMappings converts UID/GID mappings from an OCI spec to
// syscall.SysProcIDMap for user namespace configuration via cmd.Exec.
// If no mappings are provided then it defaults to mapping the current process'
// UID/GID.
func BuildUserNSMappings(
	specUIDMappings []specs.LinuxIDMapping,
	specGIDMappings []specs.LinuxIDMapping,
) ([]syscall.SysProcIDMap, []syscall.SysProcIDMap) {
	uidMappings := make([]syscall.SysProcIDMap, 0, max(1, len(specUIDMappings)))
	gidMappings := make([]syscall.SysProcIDMap, 0, max(1, len(specGIDMappings)))

	if len(specUIDMappings) > 0 {
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

	if len(specGIDMappings) > 0 {
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

// ContainsNamespaceType checks whether the given namespaces contain a
// namespace of the given namespaceType.
func ContainsNamespaceType(
	namespaces []specs.LinuxNamespace,
	namespaceType specs.LinuxNamespaceType,
) bool {
	return slices.ContainsFunc(
		namespaces,
		func(ns specs.LinuxNamespace) bool {
			return ns.Type == namespaceType
		},
	)
}
