package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// ErrInvalidCGroupPath is returned when an invalid cgroup path is specified.
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
			spec.Linux.CgroupsPath,
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

// DeleteCGroups deletes a cgroup based on the given state and/or spec.
func DeleteCGroups(state *specs.State, spec *specs.Spec) error {
	if isUnifiedCGroupsMode() {
		if err := deleteV2CGroups(state.ID, spec.Linux.CgroupsPath); err != nil {
			return err
		}
	} else {
		if err := deleteV1CGroups(spec.Linux.CgroupsPath); err != nil {
			return err
		}
	}

	return nil
}

func addV1CGroups(
	path string,
	resources *specs.LinuxResources,
	pid int,
) error {
	if !validateCgroupPath(path) {
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

func deleteV1CGroups(path string) error {
	if !validateCgroupPath(path) {
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

func addV2CGroups(
	containerID string,
	cgroupsPath string,
	resources *specs.LinuxResources,
	pid int,
) error {
	slice, group := buildSystemdCGroupSliceAndGroup(cgroupsPath, containerID)

	cgResources := cgroup2.ToResources(resources)

	if _, err := cgroup2.NewSystemd(slice, group, pid, cgResources); err != nil {
		return fmt.Errorf("create cgroups (id: %s): %w", containerID, err)
	}

	return nil
}

func deleteV2CGroups(containerID, cgroupsPath string) error {
	slice, group := buildSystemdCGroupSliceAndGroup(cgroupsPath, containerID)

	if _, err := os.Stat(filepath.Join("/sys/fs/cgroup", slice, group)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("stat cgroup path (id: %s): %w", containerID, err)
	}

	cg, err := cgroup2.LoadSystemd(slice, group)
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

func validateCgroupPath(path string) bool {
	if path == "" {
		return false
	}

	if strings.HasPrefix(filepath.Clean(path), "..") {
		return false
	}

	return true
}

func buildSystemdCGroupPath(cgroupsPath, containerID string) string {
	systemdCGroupPath := cgroupsPath
	if systemdCGroupPath == "" {
		if containerID == "" {
			return ""
		}

		systemdCGroupPath = containerID
	}

	if !strings.HasSuffix(systemdCGroupPath, ".slice") {
		systemdCGroupPath = fmt.Sprintf("%s.slice", systemdCGroupPath)
	}

	return systemdCGroupPath
}

func buildSystemdCGroupSliceAndGroup(
	cgroupsPath, containerID string,
) (string, string) {
	if cgroupsPath != "" && strings.Contains(cgroupsPath, ":") {
		parts := strings.SplitN(cgroupsPath, ":", 3)

		slice := parts[0]

		switch slice {
		case "":
			slice = "system.slice"
		case "-":
			slice = "/"
		}

		prefix := ""
		name := containerID

		if len(parts) >= 2 {
			prefix = parts[1]
		}

		if len(parts) >= 3 && parts[2] != "" {
			name = parts[2]
		}

		if strings.HasSuffix(name, ".slice") {
			return slice, name
		}

		if prefix != "" {
			return slice, fmt.Sprintf("%s-%s.scope", prefix, name)
		}

		return slice, fmt.Sprintf("%s.scope", name)
	}

	if containerID == "" {
		return "system.slice", ""
	}

	return "anocir.slice", fmt.Sprintf("%s.scope", containerID)
}
