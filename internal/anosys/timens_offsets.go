package anosys

import (
	"bytes"
	"fmt"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// SetTimeOffsets sets the time offsets for the time namespace.
func SetTimeOffsets(offsets map[string]specs.LinuxTimeOffset) error {
	var tos bytes.Buffer

	for clock, offset := range offsets {
		if n, err := tos.WriteString(
			fmt.Sprintf("%s %d %d\n", clock, offset.Secs, offset.Nanosecs),
		); err != nil || n == 0 {
			return fmt.Errorf("write time offsets")
		}
	}

	if err := os.WriteFile(
		"/proc/self/timens_offsets",
		tos.Bytes(),
		0o644,
	); err != nil {
		return fmt.Errorf("write timens offsets: %w", err)
	}

	return nil
}
