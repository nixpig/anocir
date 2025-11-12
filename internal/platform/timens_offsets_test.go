package platform

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestParseTimeOffset(t *testing.T) {
	scenarios := map[string]struct {
		clock  string
		offset specs.LinuxTimeOffset
		value  string
		err    error
	}{
		"test monotonic clock": {
			clock: "monotonic",
			offset: specs.LinuxTimeOffset{
				Secs:     23,
				Nanosecs: 42,
			},
			value: "monotonic 23 42\n",
			err:   nil,
		},
		"test boottime clock": {
			clock: "boottime",
			offset: specs.LinuxTimeOffset{
				Secs:     13,
				Nanosecs: 69,
			},
			value: "boottime 13 69\n",
			err:   nil,
		},
		"test empty offset": {
			clock:  "monotonic",
			offset: specs.LinuxTimeOffset{},
			value:  "monotonic 0 0\n",
			err:    nil,
		},
		"test invalid clock": {
			clock: "invalid",
			offset: specs.LinuxTimeOffset{
				Secs:     7,
				Nanosecs: 13,
			},
			value: "",
			err:   ErrInvalidClock,
		},
		"test empty clock": {
			clock: "",
			offset: specs.LinuxTimeOffset{
				Secs:     7,
				Nanosecs: 13,
			},
			value: "",
			err:   ErrInvalidClock,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			timeOffset, err := parseTimeOffset(data.offset, data.clock)
			assert.Equal(t, data.err, err)
			assert.Equal(t, data.value, timeOffset)
		})
	}
}
