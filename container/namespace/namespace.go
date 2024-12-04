package namespace

import (
	"errors"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

type LinuxNamespace specs.LinuxNamespace

func (ns *LinuxNamespace) ToEnv() string {
	switch ns.Type {
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

func (ns *LinuxNamespace) ToFlag() (uintptr, error) {
	switch ns.Type {
	case specs.PIDNamespace:
		return syscall.CLONE_NEWPID, nil
	case specs.NetworkNamespace:
		return syscall.CLONE_NEWNET, nil
	case specs.MountNamespace:
		return syscall.CLONE_NEWNS, nil
	case specs.IPCNamespace:
		return syscall.CLONE_NEWIPC, nil
	case specs.UTSNamespace:
		return syscall.CLONE_NEWUTS, nil
	case specs.UserNamespace:
		return syscall.CLONE_NEWUSER, nil
	case specs.CgroupNamespace:
		return syscall.CLONE_NEWCGROUP, nil
	case specs.TimeNamespace:
		return syscall.CLONE_NEWTIME, nil
	default:
		return 0, errors.New("unknown namespace type")
	}
}
