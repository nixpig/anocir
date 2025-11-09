package platform

import (
	"errors"
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var ErrIOPrioMapping = errors.New("ioprio mapping failed")

var ioprioClassMapping = map[specs.IOPriorityClass]int{
	specs.IOPRIO_CLASS_RT:   1,
	specs.IOPRIO_CLASS_BE:   2,
	specs.IOPRIO_CLASS_IDLE: 3,
}

// SetIOPriority sets the I/O priority for the current (container) process.
func SetIOPriority(ioprio *specs.LinuxIOPriority) error {
	i, err := ioprioToInt(ioprio)
	if err != nil {
		return fmt.Errorf("ioprio to int: %w", err)
	}

	if _, _, errno := unix.Syscall(unix.SYS_IOPRIO_SET, 1, 0, uintptr(i)); errno != 0 {
		return fmt.Errorf("set io priority: %w", errno)
	}

	return nil
}

func ioprioToInt(iop *specs.LinuxIOPriority) (int, error) {
	class, ok := ioprioClassMapping[iop.Class]
	if !ok {
		return 0, fmt.Errorf(
			"%w: unknown class %s",
			ErrIOPrioMapping,
			iop.Class,
		)
	}

	ioprio := (class << 13) | iop.Priority

	return ioprio, nil
}
