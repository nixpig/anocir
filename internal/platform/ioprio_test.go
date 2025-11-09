package platform

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestIOPrioToInt(t *testing.T) {
	scenarios := map[string]struct {
		ioPrio *specs.LinuxIOPriority
		value  int
		err    error
	}{
		"test realtime class": {
			ioPrio: &specs.LinuxIOPriority{
				Class:    specs.IOPRIO_CLASS_RT,
				Priority: 4,
			},
			value: 8196,
			err:   nil,
		},
		"test best-effort class": {
			ioPrio: &specs.LinuxIOPriority{
				Class:    specs.IOPRIO_CLASS_BE,
				Priority: 7,
			},
			value: 16391,
			err:   nil,
		},
		"test idle class": {
			ioPrio: &specs.LinuxIOPriority{
				Class:    specs.IOPRIO_CLASS_IDLE,
				Priority: 0,
			},
			value: 24576,
			err:   nil,
		},
		"test invalid": {
			ioPrio: &specs.LinuxIOPriority{
				Class:    "",
				Priority: 0,
			},
			value: 0,
			err:   ErrIOPrioMapping,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			ioprio, err := ioprioToInt(data.ioPrio)

			assert.ErrorIs(t, err, data.err)
			assert.Equal(t, data.value, ioprio)
		})
	}
}
