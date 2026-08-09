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
	if len(c.Spec.Linux.Sysctl) > 0 {
		if err := platform.SetSysctl(c.Spec.Linux.Sysctl); err != nil {
			return fmt.Errorf("set sysctl: %w", err)
		}
	}

	if err := platform.MountMaskedPaths(c.Spec.Linux.MaskedPaths); err != nil {
		return fmt.Errorf("mount masked paths: %w", err)
	}

	if err := platform.MountReadonlyPaths(c.Spec.Linux.ReadonlyPaths); err != nil {
		return fmt.Errorf("mount readonly paths: %w", err)
	}

	if err := platform.SetRootfsMountPropagation(
		c.Spec.Linux.RootfsPropagation,
	); err != nil {
		return fmt.Errorf("set rootfs mount propagation: %w", err)
	}

	if c.Spec.Root.Readonly {
		if err := platform.MountRootReadonly(); err != nil {
			return fmt.Errorf("remount root as readonly: %w", err)
		}
	}

	// Only set hostname if we created a new UTS namespace (Path is empty).
	// If we're joining an existing UTS namespace (Path is not empty), the
	// hostname was already set by the namespace creator.
	if slices.ContainsFunc(c.Spec.Linux.Namespaces, func(ns specs.LinuxNamespace) bool {
		return ns.Type == specs.UTSNamespace && ns.Path == ""
	}) {
		if err := unix.Sethostname([]byte(c.Spec.Hostname)); err != nil {
			return fmt.Errorf("set hostname: %w", err)
		}

		if err := unix.Setdomainname([]byte(c.Spec.Domainname)); err != nil {
			return fmt.Errorf("set domainname: %w", err)
		}
	}

	if err := platform.SetRlimits(c.Spec.Process.Rlimits); err != nil {
		return fmt.Errorf("set rlimits: %w", err)
	}

	if c.Spec.Process.Scheduler != nil {
		schedAttr, err := platform.NewSchedAttr(c.Spec.Process.Scheduler)
		if err != nil {
			return fmt.Errorf("new sched attr: %w", err)
		}

		if err := platform.SchedSetAttr(schedAttr); err != nil {
			return fmt.Errorf("set sched attr: %w", err)
		}
	}

	if c.Spec.Process.IOPriority != nil {
		ioprio, err := platform.IOPrioToInt(c.Spec.Process.IOPriority)
		if err != nil {
			return fmt.Errorf("convert ioprio to int: %w", err)
		}

		if err := platform.IOPrioSet(ioprio); err != nil {
			return fmt.Errorf("set ioprio: %w", err)
		}
	}

	if err := platform.ApplyProcessSecurity(&platform.ProcessSecurity{
		User:            &c.Spec.Process.User,
		Capabilities:    c.Spec.Process.Capabilities,
		Seccomp:         c.Spec.Linux.Seccomp,
		NoNewPrivs:      c.Spec.Process.NoNewPrivileges,
		AppArmorProfile: c.Spec.Process.ApparmorProfile,
		ProcessLabel:    c.Spec.Process.SelinuxLabel,
	}); err != nil {
		return fmt.Errorf("apply process security: %w", err)
	}

	return nil
}
