package container

import (
	"fmt"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) setupPrePivot() error {
	if err := platform.MountRootfs(c.rootFS()); err != nil {
		return fmt.Errorf("mount rootfs: %w", err)
	}

	if err := platform.MountProc(c.rootFS()); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	c.spec.Mounts = append(c.spec.Mounts, specs.Mount{
		Destination: "/dev/pts",
		Type:        "devpts",
		Source:      "devpts",
		Options: []string{
			"nosuid",
			"noexec",
			"newinstance",
			"ptmxmode=0666",
			"mode=0620",
			"gid=5",
		},
	})

	if err := platform.MountSpecMounts(c.spec.Mounts, c.rootFS()); err != nil {
		return fmt.Errorf("mount spec mounts: %w", err)
	}

	if err := platform.MountDefaultDevices(c.rootFS()); err != nil {
		return fmt.Errorf("mount default devices: %w", err)
	}

	if err := platform.CreateDeviceNodes(c.spec.Linux.Devices, c.rootFS()); err != nil {
		return fmt.Errorf("mount devices from spec: %w", err)
	}

	if err := platform.CreateDefaultSymlinks(c.rootFS()); err != nil {
		return fmt.Errorf("create default symlinks: %w", err)
	}

	return nil
}
