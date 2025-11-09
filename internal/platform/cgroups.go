package platform

import (
	"fmt"

	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// IsUnifiedCGroupsMode checks if the system is running in cgroup v2 unified mode.
func IsUnifiedCGroupsMode() bool {
	return cgroups.Mode() == cgroups.Unified
}

// AddV1CGroups adds a process to a cgroup v1 hierarchy.
func AddV1CGroups(
	path string,
	resources *specs.LinuxResources,
	pid int,
) error {
	staticPath := cgroup1.StaticPath(path)

	cg, err := cgroup1.New(staticPath, resources)
	if err != nil {
		return fmt.Errorf("create cgroups (path: %s): %w", path, err)
	}

	if err := cg.Add(cgroup1.Process{Pid: pid}); err != nil {
		return fmt.Errorf("add cgroups (path: %s, pid: %d): %w", path, pid, err)
	}

	return nil
}

// DeleteV1CGroups deletes a cgroup v1 hierarchy.
func DeleteV1CGroups(path string) error {
	staticPath := cgroup1.StaticPath(path)

	cg, err := cgroup1.Load(staticPath)
	if err != nil {
		return fmt.Errorf("load cgroups (path: %s): %w", path, err)
	}

	if err := cg.Delete(); err != nil {
		return fmt.Errorf("delete cgroups (path: %s): %w", path, err)
	}

	return nil
}

// AddV2CGroups adds a process to a cgroup v2 hierarchy.
func AddV2CGroups(
	containerID string,
	resources *specs.LinuxResources,
	pid int,
) error {
	systemdGroup := fmt.Sprintf("%s.slice", containerID)
	cgResources := cgroup2.ToResources(resources)

	cg, err := cgroup2.NewSystemd("/", systemdGroup, -1, cgResources)
	if err != nil {
		return fmt.Errorf("create cgroups (id: %s): %w", containerID, err)
	}

	if err := cg.AddProc(uint64(pid)); err != nil {
		return fmt.Errorf("add pid to cgroup2: %w", err)
	}

	return nil
}

// DeleteV2CGroups deletes a cgroup v2 hierarchy.
func DeleteV2CGroups(containerID string) error {
	systemdGroup := fmt.Sprintf("%s.slice", containerID)

	cg, err := cgroup2.LoadSystemd("/", systemdGroup)
	if err != nil {
		return fmt.Errorf("load cgroups (id: %s): %w", containerID, err)
	}

	if err := cg.Kill(); err != nil {
		return fmt.Errorf(
			"kill cgroups processes (id: %s): %w",
			containerID,
			err,
		)
	}

	if err := cg.DeleteSystemd(); err != nil {
		return fmt.Errorf("delete cgroups (id: %s): %w", containerID, err)
	}

	return nil
}
