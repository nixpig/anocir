package platform

import (
	"fmt"
	"log/slog"
	"os"
)

const (
	selinuxEnabled        = "/sys/fs/selinux/enforce"
	selinuxExecPath       = "/proc/self/attr/selinux/exec"
	selinuxLegacyExecPath = "/proc/self/attr/exec"
)

// IsSELinuxEnabled checks if SELinux is enabled on the system.
func IsSELinuxEnabled() bool {
	if _, err := os.Stat(selinuxEnabled); os.IsNotExist(err) {
		return false
	}

	return true
}

// ApplySELinuxProfile applies the given SELinux profile label to the current process.
func ApplySELinuxProfile(label string) error {
	if label == "" {
		return nil
	}

	if err := os.WriteFile(selinuxExecPath, []byte(label), 0); err != nil {
		slog.Debug("falling back to SELinux legacy path", "err", err)
		if err := os.WriteFile(selinuxLegacyExecPath, []byte(label), 0); err != nil {
			return fmt.Errorf("apply SELinux profile: %w", err)
		}
	}

	return nil
}
