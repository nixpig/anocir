package container

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"syscall"

	"github.com/nixpig/brownie/container/capabilities"
	"github.com/nixpig/brownie/container/cgroups"
	"github.com/nixpig/brownie/container/filesystem"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Reexec2(log *zerolog.Logger) error {
	if err := filesystem.PivotRoot(c.Rootfs()); err != nil {
		log.Error().Err(err).Msg("pivot root")
		return err
	}

	if err := filesystem.MountMaskedPaths(
		c.Spec.Linux.MaskedPaths,
	); err != nil {
		log.Error().Err(err).Msg("mount masked paths")
		return err
	}

	if err := filesystem.MountReadonlyPaths(
		c.Spec.Linux.ReadonlyPaths,
	); err != nil {
		log.Error().Err(err).Msg("mount readonly paths")
		return err
	}

	if c.Spec.Linux.RootfsPropagation != "" {
		if err := syscall.Mount("", "/", "", filesystem.MountOptions[c.Spec.Linux.RootfsPropagation].Flag, ""); err != nil {
			log.Error().Err(err).Msg("mount rootfs RootfsPropagation paths")
			return err
		}
	}

	if c.Spec.Root.Readonly {
		if err := syscall.Mount("", "/", "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
			log.Error().Err(err).Msg("mount rootfs readonly paths")
			return err
		}
	}

	if slices.ContainsFunc(
		c.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	) {
		if err := syscall.Sethostname(
			[]byte(c.Spec.Hostname),
		); err != nil {
			log.Error().Err(err).Msg("set hostname")
			return err
		}

		if err := syscall.Setdomainname(
			[]byte(c.Spec.Domainname),
		); err != nil {
			log.Error().Err(err).Msg("set domainname")
			return err
		}
	}

	if c.Spec.Process != nil {
		if c.Spec.Process.Rlimits != nil {
			if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
				log.Error().Err(err).Msg("set rlimits")
				return err
			}
		}

		if c.Spec.Process.Capabilities != nil {
			if err := capabilities.SetCapabilities(
				c.Spec.Process.Capabilities,
			); err != nil {
				log.Error().Err(err).Msg("set caps")
				return err
			}
		}
	}

	log.Info().Any("args", c.Spec.Process.Args).Msg("setting up command")
	cmd := exec.Command(
		c.Spec.Process.Args[0],
		c.Spec.Process.Args[1:]...,
	)

	cmd.Dir = c.Spec.Process.Cwd

	var ambientCapsFlags []uintptr
	if c.Spec.Process != nil &&
		c.Spec.Process.Capabilities != nil {
		for _, cap := range c.Spec.Process.Capabilities.Ambient {
			ambientCapsFlags = append(
				ambientCapsFlags,
				uintptr(capabilities.Capabilities[cap]),
			)
		}
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps: ambientCapsFlags,
		Credential: &syscall.Credential{
			Uid:    c.Spec.Process.User.UID,
			Gid:    c.Spec.Process.User.GID,
			Groups: c.Spec.Process.User.AdditionalGids,
		},
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := c.ExecHooks("startContainer"); err != nil {
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	// we can't get logs or anything past this point
	log.Info().Any("args", c.Spec.Process.Args).Msg("💚  --- reexec2 ---")
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("(run) error executing in reexec2")
		return err
	}

	return nil
}
