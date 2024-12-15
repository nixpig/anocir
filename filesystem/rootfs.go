package filesystem

import (
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func SetupRootfs(rootfs string, spec *specs.Spec) error {
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

func SetRootfsMountPropagation(prop string) error {
	if prop == "" {
		return nil
	}

	if err := syscall.Mount(
		"",
		"/",
		"",
		MountOptions[prop].Flag,
		"",
	); err != nil {
		return fmt.Errorf("set rootfs mount propagation (%s): %w", prop, err)
	}

	return nil
}

func MountRootReadonly(ro bool) error {
	if !ro {
		return nil
	}

	if err := syscall.Mount(
		"",
		"/",
		"",
		syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY,
		"",
	); err != nil {
		return fmt.Errorf("remount root as readonly: %w", err)
	}

	return nil
}
