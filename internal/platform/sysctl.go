package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SetSysctl sets the sysctls kernel parameters for the container process.
func SetSysctl(sysctls map[string]string) error {
	for k, v := range sysctls {
		if err := os.WriteFile(sysctlPath(k), []byte(v), 0o644); err != nil {
			return fmt.Errorf("write sysctl (%s: %s): %w", k, v, err)
		}
	}

	return nil
}

// sysctlPath converts a sysctl string to its path in /proc/sys.
func sysctlPath(sysctl string) string {
	return filepath.Join("/proc/sys", strings.ReplaceAll(sysctl, ".", "/"))
}
