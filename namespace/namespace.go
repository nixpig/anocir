package namespace

import (
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog/log"
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
		return syscall.CLONE_NEWPID
	case specs.NetworkNamespace:
		return syscall.CLONE_NEWNET
	case specs.MountNamespace:
		return syscall.CLONE_NEWNS
	case specs.IPCNamespace:
		return syscall.CLONE_NEWIPC
	case specs.UTSNamespace:
		return syscall.CLONE_NEWUTS
	case specs.UserNamespace:
		return syscall.CLONE_NEWUSER
	case specs.CgroupNamespace:
		return syscall.CLONE_NEWCGROUP
	case specs.TimeNamespace:
		return syscall.CLONE_NEWTIME
	default:
		return 0
	}
}

func (ns *LinuxNamespace) Enter() error {
	fd, err := syscall.Open(ns.Path, syscall.O_RDONLY, 0666)
	if err != nil {
		log.Error().Err(err).Str("path", ns.Path).Str("type", string(ns.Type)).Msg("failed to open namespace path")
		return fmt.Errorf("open ns path: %w", err)
	}
	defer syscall.Close(fd)

	_, _, errno := syscall.RawSyscall(unix.SYS_SETNS, uintptr(fd), 0, 0)
	if errno != 0 {
		log.Error().Str("path", ns.Path).Int("errno", int(errno)).Msg("FAIELD THE RAWSYSCALL")
		return fmt.Errorf("errno: %w", err)
	}

	return nil
}
