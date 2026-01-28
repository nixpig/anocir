package container

import (
	"fmt"
	"slices"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// setupPrePivot performs configuration of the container environment before
// pivot_root.
func (c *Container) setupPrePivot() error {
	hasMountNamespace := platform.ContainsNamespaceType(
		c.spec.Linux.Namespaces,
		specs.MountNamespace,
	)
	if hasMountNamespace {
		if err := platform.MountRootfs(c.rootFS()); err != nil {
			return fmt.Errorf("mount rootfs: %w", err)
		}
	}

	if err := platform.MountProc(c.rootFS()); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	mounts := slices.Concat(c.spec.Mounts, []specs.Mount{{
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
	}})

	if err := platform.MountSpecMounts(mounts, c.rootFS()); err != nil {
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
