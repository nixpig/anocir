package platform

import (
	"fmt"
	"os"
	"strings"
)

const (
	appArmorEnabled = "/sys/module/apparmor/parameters/enabled"
	appArmorApply   = "/proc/self/attr/apparmor/exec"
)

// IsAppArmorEnabled checks if AppArmor is enabled on the system.
func IsAppArmorEnabled() bool {
	data, err := os.ReadFile(appArmorEnabled)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(data)) == "Y"
}

// ApplyAppArmorProfile applies the given AppArmor profile to the current
// process. The profile should be in the format "profile_name" or
// "localhost/profile_name".
func ApplyAppArmorProfile(profile string) error {
	if profile == "" || profile == "unconfined" {
		return nil
	}

	profile = strings.TrimPrefix(profile, "localhost/")

	// Use exec for applying on next exec.
	profileData := fmt.Sprintf("exec %s", profile)

	if err := os.WriteFile(appArmorApply, []byte(profileData), 0); err != nil {
		legacyPath := "/proc/self/attr/exec"
		if err := os.WriteFile(legacyPath, []byte(profileData), 0); err != nil {
			return fmt.Errorf("apply apparmor profile: %w", err)
		}
	}

	return nil
}
