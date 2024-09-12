package internal

import (
	"errors"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func NamespaceToFlag(namespace specs.LinuxNamespaceType) (int, error) {
	switch namespace {
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
		return -1, errors.New("unknown namespace")
	}
}

func NamespacesToFlag(namespaces []specs.LinuxNamespace) (*uintptr, error) {
	var flags uintptr
	for _, ns := range namespaces {
		f, err := NamespaceToFlag(ns.Type)
		if err != nil {
			return nil, err
		}

		flags = flags | uintptr(f)
	}

	return &flags, nil
}
