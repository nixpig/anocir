package container

import (
	"fmt"
	"slices"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// setupPostPivot performs configuration of the container environment after
// pivot_root.
func (c *Container) setupPostPivot() error {
	if len(c.spec.Linux.Sysctl) > 0 {
		if err := platform.SetSysctl(c.spec.Linux.Sysctl); err != nil {
			return fmt.Errorf("set sysctl: %w", err)
		}
	}

	if err := platform.MountMaskedPaths(c.spec.Linux.MaskedPaths); err != nil {
		return fmt.Errorf("mount masked paths: %w", err)
	}

	if err := platform.MountReadonlyPaths(c.spec.Linux.ReadonlyPaths); err != nil {
		return fmt.Errorf("mount readonly paths: %w", err)
	}

	if err := platform.SetRootfsMountPropagation(
		c.spec.Linux.RootfsPropagation,
	); err != nil {
		return fmt.Errorf("set rootfs mount propagation: %w", err)
	}

	if c.spec.Root.Readonly {
		if err := platform.MountRootReadonly(); err != nil {
			return fmt.Errorf("remount root as readonly: %w", err)
		}
	}

	// Only set hostname if we created a new UTS namespace (Path is empty).
	// If we're joining an existing UTS namespace (Path is set), the hostname
	// was already set by the namespace creator.
	createdUTSNamespace := platform.HasNewNamespace(
		c.spec.Linux.Namespaces,
		specs.UTSNamespace,
	)

	if createdUTSNamespace {
		if err := unix.Sethostname([]byte(c.spec.Hostname)); err != nil {
			return fmt.Errorf("set hostname: %w", err)
		}

		if err := unix.Setdomainname([]byte(c.spec.Domainname)); err != nil {
			return fmt.Errorf("set domainname: %w", err)
		}
	}

	if err := platform.SetRlimits(c.spec.Process.Rlimits); err != nil {
		return fmt.Errorf("set rlimits: %w", err)
	}

	if c.spec.Process.Scheduler != nil {
		schedAttr, err := platform.NewSchedAttr(c.spec.Process.Scheduler)
		if err != nil {
			return fmt.Errorf("new sched attr: %w", err)
		}

		if err := platform.SchedSetAttr(schedAttr); err != nil {
			return fmt.Errorf("set sched attr: %w", err)
		}
	}

	if c.spec.Process.IOPriority != nil {
		ioprio, err := platform.IOPrioToInt(c.spec.Process.IOPriority)
		if err != nil {
			return fmt.Errorf("convert ioprio to int: %w", err)
		}

		if err := platform.IOPrioSet(ioprio); err != nil {
			return fmt.Errorf("set ioprio: %w", err)
		}
	}

	// When NoNewPrivileges is false, we load seccomp BEFORE dropping
	// capabilities because seccomp filter loading is a privileged operation that
	// requires CAP_SYS_ADMIN when NO_NEW_PRIVS is not set.
	//
	// See: https://man7.org/linux/man-pages/man2/seccomp.2.html
	if c.spec.Linux.Seccomp != nil && !c.spec.Process.NoNewPrivileges {
		if err := platform.LoadSeccompFilter(c.spec.Linux.Seccomp); err != nil {
			return fmt.Errorf("load seccomp filter (privileged): %w", err)
		}
	}

	if c.spec.Process.Capabilities != nil {
		if err := platform.DropBoundingCapabilities(c.spec.Process.Capabilities); err != nil {
			return fmt.Errorf("drop bounding caps: %w", err)
		}
	}

	if c.spec.Process.User.UID != 0 && c.spec.Process.Capabilities != nil {
		if err := platform.SetKeepCaps(1); err != nil {
			return fmt.Errorf("set KEEPCAPS: %w", err)
		}
	}

	if err := platform.SetUser(&c.spec.Process.User); err != nil {
		return fmt.Errorf("set user: %w", err)
	}

	if c.spec.Process.Capabilities != nil {
		b := c.spec.Process.Capabilities.Bounding
		isPrivileged := slices.ContainsFunc(b, func(s string) bool {
			return s == "ALL" || s == "CAP_ALL"
		})

		if isPrivileged {
			if err := platform.SetAllCapabilities(); err != nil {
				return fmt.Errorf("set all capabilities: %w", err)
			}
		} else {
			if err := platform.SetCapabilities(c.spec.Process.Capabilities); err != nil {
				return fmt.Errorf("set capabilities: %w", err)
			}

			if c.spec.Process.User.UID != 0 {
				if err := platform.SetKeepCaps(0); err != nil {
					return fmt.Errorf("clear KEEPCAPS: %w", err)
				}
			}
		}
	}

	if c.spec.Process.NoNewPrivileges {
		if err := platform.SetNoNewPrivs(); err != nil {
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	// When NoNewPrivileges is true, we load seccomp AFTER setting NO_NEW_PRIVS,
	// as close to execve as possible to minimize the syscall surface. The
	// NO_NEW_PRIVS bit allows unprivileged seccomp filter loading.
	if c.spec.Linux.Seccomp != nil && c.spec.Process.NoNewPrivileges {
		if err := platform.LoadSeccompFilter(c.spec.Linux.Seccomp); err != nil {
			return fmt.Errorf("load seccomp filter (unprivileged): %w", err)
		}
	}

	// Apply AppArmor profile if specified
	if c.spec.Process != nil && c.spec.Process.ApparmorProfile != "" {
		if err := platform.ApplyAppArmorProfile(c.spec.Process.ApparmorProfile); err != nil {
			return fmt.Errorf("apply apparmor profile: %w", err)
		}
	}

	return nil
}
