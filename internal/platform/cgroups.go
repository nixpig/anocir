package platform

import (
	"fmt"
	"strings"

	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// IsUnifiedCgroupsMode checks whether unified mode (i.e. cgroups v2) is
// running on the host.
func IsUnifiedCgroupsMode() bool {
	return cgroups.Mode() == cgroups.Unified
}

// CreateCgroup uses the given cgroupsPath and containerID to create a systemd
// cgroup slice and group, placing the process specified by containerPID into
// it and applying the resource restrictions from the given resources.
func CreateCgroup(
	cgroupsPath, containerID string,
	containerPID int,
	resources *specs.LinuxResources,
) error {
	slice, group := buildSystemdCGroupSliceAndGroup(
		cgroupsPath,
		containerID,
	)

	if err := validateCgroupsSliceGroup(slice, group); err != nil {
		return fmt.Errorf("validate cgroup slice and group: %w", err)
	}

	cgResources := &cgroup2.Resources{}
	if resources != nil {
		cgResources = cgroup2.ToResources(resources)
	}

	_, err := cgroup2.NewSystemd(slice, group, containerPID, cgResources)
	return err
}

// DeleteCgroup loads the cgroup manager using the given cgroupsPath and
// containerID, and deletes the corresponding cgroup.
func DeleteCgroup(cgroupsPath, containerID string) error {
	cg, err := loadCgroupManager(cgroupsPath, containerID)
	if err != nil {
		return fmt.Errorf("load cgroup manager: %w", err)
	}

	// TODO: Consider logging errors in future. Ignoring for now, as best-effort.
	_ = cg.Freeze()
	_ = cg.Kill()

	return cg.DeleteSystemd()
}

// UpdateCgroup loads the cgroup manager using the given cgroupsPath and
// containerID, and applies the given resources restrictions.
func UpdateCgroup(
	cgroupsPath, containerID string,
	resources *specs.LinuxResources,
) error {
	cg, err := loadCgroupManager(cgroupsPath, containerID)
	if err != nil {
		return fmt.Errorf("load cgroup manager: %w", err)
	}

	cgResources := &cgroup2.Resources{}
	if resources != nil {
		cgResources = cgroup2.ToResources(resources)
	}

	return cg.Update(cgResources)
}

// FreezeCgroup loads the cgroup manager using the given cgroupsPath and
// containerID, and freezes it.
func FreezeCgroup(cgroupsPath, containerID string) error {
	cg, err := loadCgroupManager(cgroupsPath, containerID)
	if err != nil {
		return fmt.Errorf("load cgroup manager: %w", err)
	}

	return cg.Freeze()
}

// ThawCgroup loads the cgroup manager using the given cgroupsPath and
// containerID, and thaws it.
func ThawCgroup(cgroupsPath, containerID string) error {
	cg, err := loadCgroupManager(cgroupsPath, containerID)
	if err != nil {
		return fmt.Errorf("load cgroup manager: %w", err)
	}

	return cg.Thaw()
}

// GetCgroupProcesses loads the cgroup manager using the given cgroupsPath and
// containerID, and returns a list of the contained process IDs
func GetCgroupProcesses(cgroupsPath, containerID string) ([]int, error) {
	cg, err := loadCgroupManager(cgroupsPath, containerID)
	if err != nil {
		return nil, fmt.Errorf("load cgroup manager: %w", err)
	}

	cgProcesses, err := cg.Procs(true)
	if err != nil {
		return nil, fmt.Errorf("load cgroup2 processes: %w", err)
	}

	var processes []int
	for _, p := range cgProcesses {
		processes = append(processes, int(p))
	}

	return processes, nil
}

func loadCgroupManager(
	cgroupsPath, containerID string,
) (*cgroup2.Manager, error) {
	slice, group := buildSystemdCGroupSliceAndGroup(cgroupsPath, containerID)

	if err := validateCgroupsSliceGroup(slice, group); err != nil {
		return nil, fmt.Errorf("validate cgroup slice and group: %w", err)
	}

	// LoadSystemd always returns a nil error.
	cg, _ := cgroup2.LoadSystemd(slice, group)

	return cg, nil
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

func validateCgroupsSliceGroup(slice, group string) error {
	for _, s := range []string{slice, group} {
		if s == "" {
			continue
		}

		if strings.Contains(s, "..") {
			return fmt.Errorf("%s contains directory traversal", s)
		}

		if strings.ContainsAny(s, "/\\") {
			return fmt.Errorf("%s contains path separator", s)
		}
	}

	return nil
}
