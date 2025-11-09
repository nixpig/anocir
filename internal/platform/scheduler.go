package platform

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var schedulerFlags = map[specs.LinuxSchedulerFlag]int{
	specs.SchedFlagResetOnFork:  0x01,
	specs.SchedFlagReclaim:      0x02,
	specs.SchedFlagDLOverrun:    0x04,
	specs.SchedFlagKeepPolicy:   0x08,
	specs.SchedFlagKeepParams:   0x10,
	specs.SchedFlagUtilClampMin: 0x20,
	specs.SchedFlagUtilClampMax: 0x40,
}

var schedulerPolicies = map[specs.LinuxSchedulerPolicy]int{
	specs.SchedOther:    0,
	specs.SchedFIFO:     1,
	specs.SchedRR:       2,
	specs.SchedBatch:    3,
	specs.SchedISO:      4,
	specs.SchedIdle:     5,
	specs.SchedDeadline: 6,
}

// SetSchedAttrs sets the scheduler attributes for the current (container)
// process.
func SetSchedAttrs(scheduler *specs.Scheduler) error {
	policy, ok := schedulerPolicies[scheduler.Policy]
	if !ok {
		return fmt.Errorf("unknown scheduler policy '%d'", policy)
	}

	flags := schedulerFlagsToInt(scheduler.Flags)

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

func schedulerFlagsToInt(flags []specs.LinuxSchedulerFlag) int {
	var f int

	for _, flag := range flags {
		f |= schedulerFlags[flag]
	}

	return f
}
