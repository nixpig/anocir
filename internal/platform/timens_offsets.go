package platform

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
)

var ErrInvalidClock = errors.New("invalid clock")

// SetTimeOffsets sets the time offsets for the time namespace.
func SetTimeOffsets(offsets map[string]specs.LinuxTimeOffset) error {
	var tos bytes.Buffer

	for clock, offset := range offsets {
		timeOffset, err := parseTimeOffset(offset, clock)
		if err != nil {
			return fmt.Errorf("parse time offset: %w", err)
		}

		if _, err := tos.WriteString(timeOffset); err != nil {
			return fmt.Errorf("write time offset: %w", err)
		}
	}

	if err := os.WriteFile("/proc/self/timens_offsets", tos.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write timens offsets: %w", err)
	}

	return nil
}

func parseTimeOffset(offset specs.LinuxTimeOffset, clock string) (string, error) {
	if clock != "monotonic" && clock != "boottime" {
		return "", ErrInvalidClock
	}

	return fmt.Sprintf("%s %d %d\n", clock, offset.Secs, offset.Nanosecs), nil
}
