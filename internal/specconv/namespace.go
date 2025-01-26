package specconv

import (
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func NamespaceTypeToFlag(nsType specs.LinuxNamespaceType) uintptr {
	switch nsType {
	case specs.PIDNamespace:
		return unix.CLONE_NEWPID
	case specs.NetworkNamespace:
		return unix.CLONE_NEWNET
	case specs.MountNamespace:
		return unix.CLONE_NEWNS
	case specs.IPCNamespace:
		return unix.CLONE_NEWIPC
	case specs.UTSNamespace:
		return unix.CLONE_NEWUTS
	case specs.UserNamespace:
		return unix.CLONE_NEWUSER
	case specs.CgroupNamespace:
		return unix.CLONE_NEWCGROUP
	case specs.TimeNamespace:
		return unix.CLONE_NEWTIME
	default:
		return 0
	}
}

func NamespaceTypeToEnv(nsType specs.LinuxNamespaceType) string {
	switch nsType {
	case specs.PIDNamespace:
		return "pid"
	case specs.NetworkNamespace:
		return "net"
	case specs.MountNamespace:
		return "mnt"
	case specs.IPCNamespace:
		return "ipc"
	case specs.UTSNamespace:
		return "uts"
	case specs.UserNamespace:
		return "user"
	case specs.CgroupNamespace:
		return "cgroup"
	case specs.TimeNamespace:
		return "time"
	default:
		return ""
	}
}
