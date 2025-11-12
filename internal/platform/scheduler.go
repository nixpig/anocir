package platform

import (
	"errors"
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// ErrUnknownSchedulerPolicy is returned when the scheduler policy is not
// recognised.
var ErrUnknownSchedulerPolicy = errors.New("unknown scheuler policy")

// schedulerFlags maps scheduler flags to their correxponding kernel values.
var schedulerFlags = map[specs.LinuxSchedulerFlag]int{
	specs.SchedFlagResetOnFork:  0x01,
	specs.SchedFlagReclaim:      0x02,
	specs.SchedFlagDLOverrun:    0x04,
	specs.SchedFlagKeepPolicy:   0x08,
	specs.SchedFlagKeepParams:   0x10,
	specs.SchedFlagUtilClampMin: 0x20,
	specs.SchedFlagUtilClampMax: 0x40,
}

// schedulerPolicies maps scheduler policies to their corresponding kernel
// values.
var schedulerPolicies = map[specs.LinuxSchedulerPolicy]int{
	specs.SchedOther:    0,
	specs.SchedFIFO:     1,
	specs.SchedRR:       2,
	specs.SchedBatch:    3,
	specs.SchedISO:      4,
	specs.SchedIdle:     5,
	specs.SchedDeadline: 6,
}

// NewSchedAttr creates a unix.SchedAttr with the scheduling attributes from
// the given scheduler or returns an error if the scheduler policy is unknown.
func NewSchedAttr(scheduler *specs.Scheduler) (*unix.SchedAttr, error) {
	policy, ok := schedulerPolicies[scheduler.Policy]
	if !ok {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrUnknownSchedulerPolicy,
			scheduler.Policy,
		)
	}

	var flags int

	for _, flag := range scheduler.Flags {
		flags |= schedulerFlags[flag]
	}

	return &unix.SchedAttr{
		Deadline: scheduler.Deadline,
		Flags:    uint64(flags),
		Size:     unix.SizeofSchedAttr,
		Nice:     scheduler.Nice,
		Period:   scheduler.Period,
		Policy:   uint32(policy),
		Priority: uint32(scheduler.Priority),
		Runtime:  scheduler.Runtime,
	}, nil
}

// SchedSetAttr sets the scheduler policy and attributes for the current
// (container) process using the given schedAttr.
func SchedSetAttr(schedAttr *unix.SchedAttr) error {
	return unix.SchedSetAttr(0, schedAttr, 0)
}
