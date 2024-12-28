package namespace

import (
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
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

func (ns *LinuxNamespace) ToFlag() uintptr {
	switch ns.Type {
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

func (ns *LinuxNamespace) Enter() error {
	fd, err := syscall.Open(ns.Path, syscall.O_RDONLY, 0666)
	if err != nil {
		return fmt.Errorf("open ns path: %w", err)
	}
	defer syscall.Close(fd)

	_, _, errno := syscall.Syscall(unix.SYS_SETNS, uintptr(fd), 0, 0)
	if errno != 0 {
		return fmt.Errorf("errno: %w", errno)
	}

	return nil
}
