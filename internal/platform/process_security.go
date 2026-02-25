package platform

import (
	"fmt"
	"slices"

	"github.com/opencontainers/runtime-spec/specs-go"
)

type ProcessSecurity struct {
	User            *specs.User
	Capabilities    *specs.LinuxCapabilities
	Seccomp         *specs.LinuxSeccomp
	NoNewPrivs      bool
	AppArmorProfile string
}

func ApplyProcessSecurity(opts *ProcessSecurity) error {
	// When NoNewPrivileges is false, we load seccomp BEFORE dropping
	// capabilities because seccomp filter loading is a privileged operation that
	// requires CAP_SYS_ADMIN when NO_NEW_PRIVS is not set.
	if opts.Seccomp != nil && !opts.NoNewPrivs {
		if err := LoadSeccompFilter(opts.Seccomp); err != nil {
			return fmt.Errorf("load seccomp filter (privileged): %w", err)
		}
	}

	if opts.Capabilities != nil {
		if err := DropBoundingCapabilities(opts.Capabilities); err != nil {
			return fmt.Errorf("drop bounding caps: %w", err)
		}
	}

	if opts.User != nil && opts.User.UID != 0 && opts.Capabilities != nil {
		if err := SetKeepCaps(1); err != nil {
			return fmt.Errorf("set KEEPCAPS 1: %w", err)
		}
	}

	if err := SetUser(opts.User); err != nil {
		return fmt.Errorf("set user: %w", err)
	}

	if opts.Capabilities != nil {
		isPrivileged := slices.ContainsFunc(opts.Capabilities.Bounding, func(s string) bool {
			return s == "ALL" || s == "CAP_ALL"
		})

		if isPrivileged {
			if err := SetAllCapabilities(); err != nil {
				return fmt.Errorf("set all capabilities: %w", err)
			}
		} else {
			if err := SetCapabilities(opts.Capabilities); err != nil {
				return fmt.Errorf("set capabilities: %w", err)
			}

			if opts.User != nil && opts.User.UID != 0 {
				if err := SetKeepCaps(0); err != nil {
					return fmt.Errorf("set KEEPCAPS 0: %w", err)
				}
			}
		}
	}

	if opts.NoNewPrivs {
		if err := SetNoNewPrivs(); err != nil {
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	// When NoNewPrivileges is true, we load seccomp AFTER setting NO_NEW_PRIVS,
	// as close to execve as possible to minimize the syscall surface. The
	// NO_NEW_PRIVS bit allows unprivileged seccomp filter loading.
	if opts.Seccomp != nil && opts.NoNewPrivs {
		if err := LoadSeccompFilter(opts.Seccomp); err != nil {
			return fmt.Errorf("load seccomp filter: %w", err)
		}
	}

	if opts.AppArmorProfile != "" {
		if err := ApplyAppArmorProfile(opts.AppArmorProfile); err != nil {
			return fmt.Errorf("apply apparmor profile: %w", err)
		}
	}

	return nil
}
