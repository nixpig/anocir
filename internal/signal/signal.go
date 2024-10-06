package signal

import (
	"fmt"
	"syscall"
)

func FromInt(s int) (syscall.Signal, error) {
	switch s {
	case 1:
		return syscall.SIGHUP, nil
	case 2:
		return syscall.SIGINT, nil
	case 3:
		return syscall.SIGQUIT, nil
	case 6:
		return syscall.SIGABRT, nil
	case 9:
		return syscall.SIGKILL, nil
	case 15:
		return syscall.SIGTERM, nil
	case 17:
		return syscall.SIGCHLD, nil
	case 19, 20, 21, 22:
		return syscall.SIGSTOP, nil
	}

	return -1, fmt.Errorf("signal not recognised (%d)", s)
}

func FromString(s string) (syscall.Signal, error) {
	switch s {
	case "HUP", "SIGHUP":
		return syscall.SIGHUP, nil
	case "INT", "SIGINT":
		return syscall.SIGINT, nil
	case "QUIT", "SIGQUIT":
		return syscall.SIGQUIT, nil
	case "ABRT", "SIGABRT":
		return syscall.SIGABRT, nil
	case "KILL", "SIGKILL":
		return syscall.SIGKILL, nil
	case "TERM", "SIGTERM":
		return syscall.SIGTERM, nil
	case "CHLD", "SIGCHLD":
		return syscall.SIGCHLD, nil
	case "STOP", "SIGSTOP":
		return syscall.SIGSTOP, nil
	}

	return syscall.Signal(-1), fmt.Errorf("convert signal string to int (%s)", s)
}
