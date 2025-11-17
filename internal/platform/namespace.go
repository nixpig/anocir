package platform

import (
	"errors"
	"fmt"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

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

func SetNS(path string) error {
	fd, err := unix.Open(path, unix.O_RDONLY, 0o666)
	if err != nil {
		return fmt.Errorf("open ns path %s: %w", path, err)
	}

	_, _, errno := unix.Syscall(unix.SYS_SETNS, uintptr(fd), 0, 0)
	if errno != 0 {
		return fmt.Errorf("set namespace %s errno: %w", path, errno)
	}

	return unix.Close(fd)
}

func ValidateNSPath(ns *specs.LinuxNamespace) error {
	suffix := fmt.Sprintf("/%s", NamespaceEnvs[ns.Type])

	if ns.Type == specs.PIDNamespace {
		if !strings.HasSuffix(ns.Path, suffix) &&
			!strings.HasSuffix(ns.Path, suffix+"_for_children") {
			return ErrInvalidNamespacePath
		}
	} else if !strings.HasSuffix(ns.Path, suffix) {
		return ErrInvalidNamespacePath
	}

	return nil
}
