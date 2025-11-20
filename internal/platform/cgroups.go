package platform

import (
	"errors"
	"fmt"

	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/opencontainers/runtime-spec/specs-go"
)

var ErrInvalidCGroupPath = errors.New("invalid cgroup path")

// isUnifiedCGroupsMode checks if the system is running in cgroup v2 unified
// mode.
func isUnifiedCGroupsMode() bool {
	return cgroups.Mode() == cgroups.Unified
}

// AddCGroups creates a cgroup with the configuration from the given spec and
// adds the process from the given state to it.
func AddCGroups(state *specs.State, spec *specs.Spec) error {
	if isUnifiedCGroupsMode() {
		if err := addV2CGroups(
			state.ID,
			spec.Linux.Resources,
			state.Pid,
		); err != nil {
			return fmt.Errorf("add to v2 cgroup: %w", err)
		}
	} else {
		if err := addV1CGroups(
			spec.Linux.CgroupsPath,
			spec.Linux.Resources,
			state.Pid,
		); err != nil {
			return fmt.Errorf("add to v1 cgroup: %w", err)
		}
	}

	return nil
}

func DeleteCGroups(state *specs.State, spec *specs.Spec) error {
	if isUnifiedCGroupsMode() {
		if err := deleteV2CGroups(state.ID); err != nil {
			return err
		}
	} else {
		if err := deleteV1CGroups(spec.Linux.CgroupsPath); err != nil {
			return err
		}
	}

	return nil
}

// addV1CGroups adds a process to a cgroup v1 hierarchy.
func addV1CGroups(
	path string,
	resources *specs.LinuxResources,
	pid int,
) error {
	if path == "" {
		return ErrInvalidCGroupPath
	}

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

// deleteV1CGroups deletes a cgroup v1 hierarchy.
func deleteV1CGroups(path string) error {
	if path == "" {
		return ErrInvalidCGroupPath
	}

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

// addV2CGroups adds a process to a cgroup v2 hierarchy.
func addV2CGroups(
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

// deleteV2CGroups deletes a cgroup v2 hierarchy.
func deleteV2CGroups(containerID string) error {
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
