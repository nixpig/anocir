package filesystem

import (
	"fmt"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func SetupRootfs(root string, spec *specs.Spec, log *zerolog.Logger) error {
	rootfs := root

	if spec.Root != nil {
		rootfs = filepath.Join(root, spec.Root.Path)
	}

	log.Info().Msg("mount rootfs")
	if err := mountRootfs(rootfs); err != nil {
		return fmt.Errorf("mount rootfs: %w", err)
	}

	log.Info().Msg("mount proc")
	if err := mountProc(rootfs); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	log.Info().Msg("mount spec mounts")
	if err := mountSpecMounts(
		spec.Mounts,
		rootfs,
	); err != nil {
		return fmt.Errorf("mount spec mounts: %w", err)
	}

	log.Info().Msg("mount default devices")
	if err := mountDefaultDevices(rootfs); err != nil {
		return fmt.Errorf("mount default devices: %w", err)
	}

	log.Info().Msg("mount spec devices")
	if err := mountSpecDevices(
		spec.Linux.Devices,
		rootfs,
		log,
	); err != nil {
		return fmt.Errorf("mount spec devices: %w", err)
	}

	log.Info().Msg("create default symlinks")
	if err := createSymlinks(
		defaultSymlinks,
		rootfs,
	); err != nil {
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
