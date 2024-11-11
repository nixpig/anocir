package container

import (
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
		return err
	}

	if err := filesystem.MountMaskedPaths(
		c.Spec.Linux.MaskedPaths,
	); err != nil {
		return err
	}

	if err := filesystem.MountReadonlyPaths(
		c.Spec.Linux.ReadonlyPaths,
	); err != nil {
		return err
	}

	if c.Spec.Linux.RootfsPropagation != "" {
		if err := syscall.Mount("", "/", "", filesystem.MountOptions[c.Spec.Linux.RootfsPropagation].Flag, ""); err != nil {
			return err
		}
	}

	if c.Spec.Root.Readonly {
		if err := syscall.Mount("", "/", "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
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
			return err
		}

		if err := syscall.Setdomainname(
			[]byte(c.Spec.Domainname),
		); err != nil {
			return err
		}
	}

	if c.Spec.Process != nil {
		if c.Spec.Process.Capabilities != nil {
			if err := capabilities.SetCapabilities(
				c.Spec.Process.Capabilities,
			); err != nil {
				return err
			}
		}

		if c.Spec.Process.Rlimits != nil {
			if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
				return err
			}
		}
	}

	cmd := exec.Command(
		c.Spec.Process.Args[0],
		c.Spec.Process.Args[1:]...,
	)

	cmd.Dir = c.Spec.Process.Cwd

	log.Info().Msg("✅ before credential set")

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

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil

}
