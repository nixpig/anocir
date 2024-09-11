package internal

import (
	"errors"
	"syscall"

	"github.com/nixpig/brownie/pkg/config"
)

func NamespaceToFlag(namespace config.NamespaceType) (int, error) {
	switch namespace {
	case config.PIDNS:
		return syscall.CLONE_NEWPID, nil
	case config.NetNS:
		return syscall.CLONE_NEWNET, nil
	case config.MountNS:
		return syscall.CLONE_NEWNS, nil
	case config.IPCNS:
		return syscall.CLONE_NEWIPC, nil
	case config.UTSNS:
		return syscall.CLONE_NEWUTS, nil
	case config.UserNS:
		return syscall.CLONE_NEWUSER, nil
	case config.CGroupNS:
		return syscall.CLONE_NEWCGROUP, nil
	case config.TimeNS:
		return syscall.CLONE_NEWTIME, nil
	default:
		return -1, errors.New("unknown namespace")
	}
}

func NamespacesToFlag(namespaces []config.Namespace) (*uintptr, error) {
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
