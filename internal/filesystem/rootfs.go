package filesystem

import "github.com/opencontainers/runtime-spec/specs-go"

func SetupRootfs(rootfs string, spec *specs.Spec) error {
	if err := mountRootfs(rootfs); err != nil {
		return err
	}

	if err := mountProc(rootfs); err != nil {
		return err
	}

	if err := mountSpecMounts(
		spec.Mounts,
		rootfs,
	); err != nil {
		return err
	}

	if err := mountDefaultDevices(rootfs); err != nil {
		return err
	}

	if err := mountSpecDevices(
		spec.Linux.Devices,
		rootfs,
	); err != nil {
		return err
	}

	if err := createSymlinks(
		defaultSymlinks,
		rootfs,
	); err != nil {
		return err
	}

	if err := pivotRootfs(rootfs); err != nil {
		return err
	}

	return nil
}
