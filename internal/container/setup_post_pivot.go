package container

import (
	"fmt"
	"slices"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func (c *Container) setupPostPivot() error {
	if c.spec.Linux.Sysctl != nil {
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

	hasUTSNamespace := slices.ContainsFunc(
		c.spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	)

	if hasUTSNamespace {
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

	if c.spec.Process.Capabilities != nil {
		if err := platform.DropBoundingCapabilities(c.spec.Process.Capabilities); err != nil {
			return fmt.Errorf("drop bounding caps: %w", err)
		}
	}

	if c.spec.Process.User.UID != 0 && c.spec.Process.Capabilities != nil {
		if err := unix.Prctl(unix.PR_SET_KEEPCAPS, 1, 0, 0, 0); err != nil {
			return fmt.Errorf("set KEEPCAPS: %w", err)
		}
	}

	if err := platform.SetUser(&c.spec.Process.User); err != nil {
		return fmt.Errorf("set user: %w", err)
	}

	if c.spec.Process.Capabilities != nil {
		if err := platform.SetCapabilities(c.spec.Process.Capabilities); err != nil {
			return fmt.Errorf("set capabilities: %w", err)
		}

		if c.spec.Process.User.UID != 0 {
			if err := unix.Prctl(unix.PR_SET_KEEPCAPS, 0, 0, 0, 0); err != nil {
				return fmt.Errorf("clear KEEPCAPS: %w", err)
			}
		}
	}

	if c.spec.Process.NoNewPrivileges {
		if err := platform.SetNoNewPrivs(); err != nil {
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	return nil
}
