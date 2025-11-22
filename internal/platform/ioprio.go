package platform

import (
	"errors"
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// ErrUnknownIOPrioClass is returned when the specified I/O priority class is
// not recognised.
var ErrUnknownIOPrioClass = errors.New("ioprio mapping failed")

// ioPrioClasses maps the I/O priority classes to the corresponding kernel
// values.
var ioPrioClasses = map[specs.IOPriorityClass]int{
	specs.IOPRIO_CLASS_RT:   1,
	specs.IOPRIO_CLASS_BE:   2,
	specs.IOPRIO_CLASS_IDLE: 3,
}

// IOPrioSet sets the I/O priority for the current (container) process.
func IOPrioSet(ioprio int) error {
	if _, _, errno := unix.Syscall(unix.SYS_IOPRIO_SET, 1, 0, uintptr(ioprio)); errno != 0 {
		return fmt.Errorf("ioprio_set: %d", errno)
	}

	return nil
}

// IOPrioToInt converts the given iop to its corresponding integer value.
func IOPrioToInt(iop *specs.LinuxIOPriority) (int, error) {
	class, ok := ioPrioClasses[iop.Class]
	if !ok {
		return 0, fmt.Errorf(
			"%w: unknown class %s",
			ErrUnknownIOPrioClass,
			iop.Class,
		)
	}

	ioprio := (class << 13) | iop.Priority

	return ioprio, nil
}
