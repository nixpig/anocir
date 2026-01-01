package platform

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// SetUser sets the user and group IDs for the current (container) process.
func SetUser(user *specs.User) error {
	if len(user.AdditionalGids) > 0 {
		additionalGids := make([]int, len(user.AdditionalGids))

		for i, gid := range user.AdditionalGids {
			additionalGids[i] = int(gid)
		}

		if err := unix.Setgroups(additionalGids); err != nil {
			return fmt.Errorf("set additional GIDs: %w", err)
		}
	}

	if err := unix.Setgid(int(user.GID)); err != nil {
		return fmt.Errorf("set GID: %w", err)
	}

	if err := unix.Setuid(int(user.UID)); err != nil {
		return fmt.Errorf("set UID: %w", err)
	}

	return nil
}
