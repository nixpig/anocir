package anosys

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// SetSysctl sets the kernel parameters (sysctls) for the container process.
func SetSysctl(sc map[string]string) error {
	for k, v := range sc {
		kp := strings.ReplaceAll(k, ".", "/")

		if err := os.WriteFile(
			path.Join("/proc/sys", kp),
			[]byte(v),
			0o644,
		); err != nil {
			return fmt.Errorf("write sysctl (%s: %s): %w", kp, v, err)
		}
	}

	return nil
}
