package filesystem

import (
	"fmt"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func SetupRootfs(root string, spec *specs.Spec) error {
	rootfs := root

	if spec.Root != nil {
		rootfs = filepath.Join(root, spec.Root.Path)
	}

	if err := mountRootfs(rootfs); err != nil {
		return fmt.Errorf("mount rootfs: %w", err)
	}

	if err := mountProc(rootfs); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	if err := mountSpecMounts(spec.Mounts, rootfs); err != nil {
		return fmt.Errorf("mount spec mounts: %w", err)
	}

	if err := mountDefaultDevices(rootfs); err != nil {
		return fmt.Errorf("mount default devices: %w", err)
	}

	if err := mountSpecDevices(spec.Linux.Devices, rootfs); err != nil {
		return fmt.Errorf("mount spec devices: %w", err)
	}

	if err := createSymlinks(defaultSymlinks, rootfs); err != nil {
		return fmt.Errorf("create symlinks: %w", err)
	}

	return nil
}

func PivotRoot(rootfs string) error {
	if err := pivotRootfs(rootfs); err != nil {
		return fmt.Errorf("pivot root: %w", err)
	}

	return nil
}
