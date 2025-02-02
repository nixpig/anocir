package anosys

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func SetSchedAttrs(scheduler *specs.Scheduler) error {
	policy, err := schedulerPolicyToInt(scheduler.Policy)
	if err != nil {
		return fmt.Errorf("scheduler policy to int: %w", err)
	}

	flags, err := schedulerFlagsToInt(scheduler.Flags)
	if err != nil {
		return fmt.Errorf("scheduler flags to int: %w", err)
	}

	schedAttr := unix.SchedAttr{
		Deadline: scheduler.Deadline,
		Flags:    uint64(flags),
		Size:     unix.SizeofSchedAttr,
		Nice:     scheduler.Nice,
		Period:   scheduler.Period,
		Policy:   uint32(policy),
		Priority: uint32(scheduler.Priority),
		Runtime:  scheduler.Runtime,
	}

	if err := unix.SchedSetAttr(0, &schedAttr, 0); err != nil {
		return fmt.Errorf("set schedattrs: %w", err)
	}

	return nil
}

func schedulerPolicyToInt(policy specs.LinuxSchedulerPolicy) (int, error) {
	switch policy {
	case specs.SchedOther:
		return 0, nil
	case specs.SchedFIFO:
		return 1, nil
	case specs.SchedRR:
		return 2, nil
	case specs.SchedBatch:
		return 3, nil
	case specs.SchedISO:
		return 4, nil
	case specs.SchedIdle:
		return 5, nil
	case specs.SchedDeadline:
		return 6, nil
	default:
		return -1, fmt.Errorf("unknown policy: %s", policy)
	}
}

func schedulerFlagsToInt(flags []specs.LinuxSchedulerFlag) (int, error) {
	var f int

	for _, flag := range flags {
		switch flag {
		case specs.SchedFlagResetOnFork:
			f |= 0x01
		case specs.SchedFlagReclaim:
			f |= 0x02
		case specs.SchedFlagDLOverrun:
			f |= 0x04
		case specs.SchedFlagKeepPolicy:
			f |= 0x08
		case specs.SchedFlagKeepParams:
			f |= 0x10
		case specs.SchedFlagUtilClampMin:
			f |= 0x20
		case specs.SchedFlagUtilClampMax:
			f |= 0x40
		default:
			return -1, fmt.Errorf("unknown flag: %s", flag)
		}
	}

	return f, nil
}
