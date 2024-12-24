package sysctl

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func SetSysctl(sc map[string]string) error {
	for k, v := range sc {
		fmt.Print(k, v)
		kp := strings.ReplaceAll(k, ".", "/")
		if err := os.WriteFile(
			path.Join("/proc/sys", kp),
			[]byte(v),
			0644,
		); err != nil {
			return fmt.Errorf("write sysctl (%s: %s): %w", kp, v, err)
		}
	}

	return nil
}
