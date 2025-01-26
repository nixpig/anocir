package iopriority

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var IOPrioClassMapping = map[specs.IOPriorityClass]int{
	specs.IOPRIO_CLASS_RT:   1,
	specs.IOPRIO_CLASS_BE:   2,
	specs.IOPRIO_CLASS_IDLE: 3,
}

func ToInt(iop *specs.LinuxIOPriority) (int, error) {
	class, ok := IOPrioClassMapping[iop.Class]
	if !ok {
		return 0, fmt.Errorf("unknown ioprio class: %s", iop.Class)
	}

	ioprio := (class << 13) | iop.Priority

	return ioprio, nil
}

func SetIOPriority(ioprio int) error {
	if _, _, errno := unix.Syscall(unix.SYS_IOPRIO_SET, 1, 0, uintptr(ioprio)); errno != 0 {
		return fmt.Errorf("set io priority: %w", errno)
	}

	return nil
}
