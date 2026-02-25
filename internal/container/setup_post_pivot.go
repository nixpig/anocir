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
	// If we're joining an existing UTS namespace (Path is not empty), the
	// hostname was already set by the namespace creator.
	if slices.ContainsFunc(c.spec.Linux.Namespaces, func(ns specs.LinuxNamespace) bool {
		return ns.Type == specs.UTSNamespace && ns.Path == ""
	}) {
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

	if err := platform.ApplyProcessSecurity(&platform.ProcessSecurity{
		User:            &c.spec.Process.User,
		Capabilities:    c.spec.Process.Capabilities,
		Seccomp:         c.spec.Linux.Seccomp,
		NoNewPrivs:      c.spec.Process.NoNewPrivileges,
		AppArmorProfile: c.spec.Process.ApparmorProfile,
	}); err != nil {
		return fmt.Errorf("apply process security: %w", err)
	}

	return nil
}
