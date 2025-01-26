package cgroups

import (
	"fmt"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func AddV2(
	containerID string,
	devices []specs.LinuxDeviceCgroup,
	pid int,
) error {
	cg, err := cgroup2.NewSystemd(
		"/",
		fmt.Sprintf("%s.slice", containerID),
		-1,
		&cgroup2.Resources{
			Devices: devices,
		},
	)
	if err != nil {
		return fmt.Errorf("create cgroups (id: %s): %w", containerID, err)
	}

	if err := cg.AddProc(uint64(pid)); err != nil {
		return fmt.Errorf("add pid to cgroup2: %w", err)
	}

	return nil
}

func DeleteV2(containerID string) error {
	cg, err := cgroup2.LoadSystemd("/", fmt.Sprintf("%s.slice", containerID))
	if err != nil {
		return fmt.Errorf("load cgroups (id: %s): %w", containerID, err)
	}

	if err := cg.Kill(); err != nil {
		return fmt.Errorf("kill cgroups processes (id: %s): %w", containerID, err)
	}

	if err := cg.DeleteSystemd(); err != nil {
		return fmt.Errorf("delete cgroups (id: %s): %w", containerID, err)
	}

	return nil
}
