package signal

import (
	"errors"
	"fmt"
	"strconv"
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

	return -1, errors.New("unhandled signal")
}

func FromString(s string) (syscall.Signal, error) {
	sig, err := strconv.Atoi(s)
	if err != nil {
		return -1, fmt.Errorf("signal string to int: %w", err)
	}

	return FromInt(sig)
}
