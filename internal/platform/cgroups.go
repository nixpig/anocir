package platform

import (
	"errors"
	"fmt"
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
	var resources *specs.LinuxResources

	if spec.Linux.Resources != nil {
		resources = spec.Linux.Resources
	}

	if isUnifiedCGroupsMode() {
		if err := addV2CGroups(
			state.ID,
			spec.Linux.CgroupsPath,
			resources,
			state.Pid,
		); err != nil {
			return fmt.Errorf("add to v2 cgroup: %w", err)
		}
	} else {
		if err := addV1CGroups(
			spec.Linux.CgroupsPath,
			resources,
			state.Pid,
		); err != nil {
			return fmt.Errorf("add to v1 cgroup: %w", err)
		}
	}

	return nil
}

// DeleteCGroups deletes a cgroup based on the given state and/or spec.
func DeleteCGroups(state *specs.State, spec *specs.Spec) error {
	// TODO: Freeze cgroups?

	if isUnifiedCGroupsMode() {
		return deleteV2CGroups(state.ID, spec.Linux.CgroupsPath)
	} else {
		return deleteV1CGroups(spec.Linux.CgroupsPath)
	}
}

func UpdateCgroups(
	state *specs.State,
	spec *specs.Spec,
	resources *specs.LinuxResources,
) error {
	if isUnifiedCGroupsMode() {
		return updateV2CGroups(state.ID, spec.Linux.CgroupsPath, resources)
	} else {
		return updateV1CGroups()
	}
}

func GetProcesses(state *specs.State, spec *specs.Spec) ([]int, error) {
	var processes []int

	if isUnifiedCGroupsMode() {
		// TODO: Figure out what to load.
		slice, group := buildSystemdCGroupSliceAndGroup(
			spec.Linux.CgroupsPath,
			state.ID,
		)

		cg, err := cgroup2.Load(filepath.Join("/", slice, group))
		if err != nil {
			return nil, fmt.Errorf("load cgroup2: %w", err)
		}

		cgProcesses, err := cg.Procs(true)
		if err != nil {
			return nil, fmt.Errorf("load cgroup2 processes: %w", err)
		}

		for _, p := range cgProcesses {
			processes = append(processes, int(p))
		}
	} else {
		staticPath := cgroup1.StaticPath(spec.Linux.CgroupsPath)
		cg, err := cgroup1.Load(staticPath)
		if err != nil {
			return nil, fmt.Errorf("load cgroup1 from path: %w", err)
		}

		// TODO: Figure out what to use as Name.
		cgProcesses, err := cg.Processes(cgroup1.Pids, true)
		if err != nil {
			return nil, fmt.Errorf("load cgroup1 processes: %w", err)
		}

		for _, p := range cgProcesses {
			processes = append(processes, p.Pid)
		}
	}

	return processes, nil
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

	// LoadSystemd always returns a nil error
	cg, _ := cgroup2.LoadSystemd(slice, group)

	// TODO: Consider logging errors in future. Ignoring for now, as best-effort.
	_ = cg.Kill()
	_ = cg.DeleteSystemd()

	return nil
}

func updateV1CGroups() error {
	return errors.New("not implemented yet")
}

func updateV2CGroups(
	containerID, cgroupsPath string,
	resources *specs.LinuxResources,
) error {
	slice, group := buildSystemdCGroupSliceAndGroup(cgroupsPath, containerID)

	// LoadSystemd always returns a nil error
	cg, _ := cgroup2.LoadSystemd(slice, group)

	cgResources := cgroup2.ToResources(resources)

	if err := cg.Update(cgResources); err != nil {
		return fmt.Errorf("update cgroup resources: %w", err)
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

	return "system.slice", fmt.Sprintf("anocir-%s.scope", containerID)
}
