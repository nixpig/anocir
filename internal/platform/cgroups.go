package platform

import (
	"fmt"
	"strings"

	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func IsUnifiedCgroupsMode() bool {
	return cgroups.Mode() == cgroups.Unified
}

func CreateCgroup(
	cgroupsPath, containerID string,
	containerPID int,
	resources *specs.LinuxResources,
) error {
	slice, group := buildSystemdCGroupSliceAndGroup(
		cgroupsPath,
		containerID,
	)

	cgResources := &cgroup2.Resources{}
	if resources != nil {
		cgResources = cgroup2.ToResources(resources)
	}

	_, err := cgroup2.NewSystemd(slice, group, containerPID, cgResources)
	return err
}

func DeleteCgroup(cgroupsPath, containerID string) error {
	cg := loadCgroupManager(cgroupsPath, containerID)

	// TODO: Consider logging errors in future. Ignoring for now, as best-effort.
	_ = cg.Freeze()
	_ = cg.Kill()
	_ = cg.DeleteSystemd()

	return nil
}

func UpdateCgroup(
	cgroupsPath, containerID string,
	resources *specs.LinuxResources,
) error {
	cg := loadCgroupManager(cgroupsPath, containerID)

	cgResources := &cgroup2.Resources{}
	if resources != nil {
		cgResources = cgroup2.ToResources(resources)
	}

	return cg.Update(cgResources)
}

func FreezeCgroup(cgroupsPath, containerID string) error {
	cg := loadCgroupManager(cgroupsPath, containerID)

	return cg.Freeze()
}

func ThawCgroup(cgroupsPath, containerID string) error {
	cg := loadCgroupManager(cgroupsPath, containerID)

	return cg.Thaw()
}

func GetCgroupProcesses(cgroupsPath, containerID string) ([]int, error) {
	cg := loadCgroupManager(cgroupsPath, containerID)

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
) *cgroup2.Manager {
	slice, group := buildSystemdCGroupSliceAndGroup(cgroupsPath, containerID)

	// LoadSystemd always returns a nil error.
	cg, _ := cgroup2.LoadSystemd(slice, group)

	return cg
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
