package scheduler

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func PolicyToInt(policy specs.LinuxSchedulerPolicy) (int, error) {
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

func FlagsToInt(flags []specs.LinuxSchedulerFlag) (int, error) {
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
