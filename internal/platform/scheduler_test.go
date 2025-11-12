package platform

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestNewSchedAttr(t *testing.T) {
	t.Run("test valid scenarios", func(t *testing.T) {
		scenarios := map[string]struct {
			scheduler *specs.Scheduler
			schedAttr *unix.SchedAttr
		}{
			"test SCHED_OTHER policy with nice": {
				scheduler: &specs.Scheduler{
					Policy: specs.SchedOther,
					Nice:   10,
				},
				schedAttr: &unix.SchedAttr{
					Policy: 0,
					Nice:   10,
					Size:   unix.SizeofSchedAttr,
				},
			},
			"test SCHED_FIFO policy with priority": {
				scheduler: &specs.Scheduler{
					Policy:   specs.SchedFIFO,
					Priority: 50,
				},
				schedAttr: &unix.SchedAttr{
					Policy:   1,
					Priority: 50,
					Size:     unix.SizeofSchedAttr,
				},
			},
			"test SCHED_RR policy": {
				scheduler: &specs.Scheduler{
					Policy: specs.SchedRR,
				},
				schedAttr: &unix.SchedAttr{
					Policy: 2,
					Size:   unix.SizeofSchedAttr,
				},
			},
			"test SCHED_BATCH policy": {
				scheduler: &specs.Scheduler{
					Policy: specs.SchedBatch,
				},
				schedAttr: &unix.SchedAttr{
					Policy: 3,
					Size:   unix.SizeofSchedAttr,
				},
			},
			"test SCHED_ISO policy": {
				scheduler: &specs.Scheduler{
					Policy: specs.SchedISO,
				},
				schedAttr: &unix.SchedAttr{
					Policy: 4,
					Size:   unix.SizeofSchedAttr,
				},
			},
			"test SCHED_IDLE policy": {
				scheduler: &specs.Scheduler{
					Policy: specs.SchedIdle,
				},
				schedAttr: &unix.SchedAttr{
					Policy: 5,
					Size:   unix.SizeofSchedAttr,
				},
			},
			"test SCHED_DEADLINE policy with all fields": {
				scheduler: &specs.Scheduler{
					Policy:   specs.SchedDeadline,
					Runtime:  1000000,
					Deadline: 2000000,
					Period:   3000000,
				},
				schedAttr: &unix.SchedAttr{
					Policy:   6,
					Runtime:  1000000,
					Deadline: 2000000,
					Period:   3000000,
					Size:     unix.SizeofSchedAttr,
				},
			},
			"test empty flags": {
				scheduler: &specs.Scheduler{
					Policy: specs.SchedOther,
					Flags:  []specs.LinuxSchedulerFlag{},
				},
				schedAttr: &unix.SchedAttr{
					Policy: 0,
					Flags:  0x0,
					Size:   unix.SizeofSchedAttr,
				},
			},
			"test single flag": {
				scheduler: &specs.Scheduler{
					Policy: specs.SchedOther,
					Flags: []specs.LinuxSchedulerFlag{
						specs.SchedFlagResetOnFork,
					},
				},
				schedAttr: &unix.SchedAttr{
					Policy: 0,
					Flags:  0x01,
					Size:   unix.SizeofSchedAttr,
				},
			},
			"test multiple flags": {
				scheduler: &specs.Scheduler{
					Policy: specs.SchedOther,
					Flags: []specs.LinuxSchedulerFlag{
						specs.SchedFlagResetOnFork,
						specs.SchedFlagReclaim,
						specs.SchedFlagDLOverrun,
						specs.SchedFlagKeepPolicy,
						specs.SchedFlagKeepParams,
						specs.SchedFlagUtilClampMin,
						specs.SchedFlagUtilClampMax,
					},
				},
				schedAttr: &unix.SchedAttr{
					Policy: 0,
					Flags:  0x7f,
					Size:   unix.SizeofSchedAttr,
				},
			},
		}

		for scenario, data := range scenarios {
			t.Run(scenario, func(t *testing.T) {
				schedAttr, err := NewSchedAttr(data.scheduler)
				assert.NoError(t, err)
				assert.Equal(t, data.schedAttr, schedAttr)
			})
		}
	})

	t.Run("test invalid scenarios", func(t *testing.T) {
		scenarios := map[string]struct {
			scheduler *specs.Scheduler
		}{
			"test unknown policy": {
				scheduler: &specs.Scheduler{
					Policy: specs.LinuxSchedulerPolicy("SCHED_UNKNOWN"),
				},
			},
			"test empty policy": {
				scheduler: &specs.Scheduler{
					Policy: specs.LinuxSchedulerPolicy(""),
				},
			},
		}

		for scenario, data := range scenarios {
			t.Run(scenario, func(t *testing.T) {
				schedAttr, err := NewSchedAttr(data.scheduler)
				assert.ErrorIs(t, err, ErrUnknownSchedulerPolicy)
				assert.Nil(t, schedAttr)
			})
		}
	})
}
