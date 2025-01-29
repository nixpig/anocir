package anosys

import (
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func SetUser(user *specs.User) error {
	if err := syscall.Setuid(int(user.UID)); err != nil {
		return fmt.Errorf("set UID: %w", err)
	}

	if err := syscall.Setgid(int(user.GID)); err != nil {
		return fmt.Errorf("set GID: %w", err)
	}

	if len(user.AdditionalGids) > 0 {
		additionalGids := make([]int, len(user.AdditionalGids))

		for i, gid := range user.AdditionalGids {
			additionalGids[i] = int(gid)
		}

		if err := syscall.Setgroups(additionalGids); err != nil {
			return fmt.Errorf("set additional GIDs: %w", err)
		}
	}

	return nil
}
