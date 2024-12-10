package container

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"syscall"

	"github.com/nixpig/brownie/capabilities"
	"github.com/nixpig/brownie/cgroups"
	"github.com/nixpig/brownie/filesystem"
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

	cloneFlags := uintptr(0)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  cloneFlags,
		AmbientCaps: ambientCapsFlags,
		Setpgid:     true,
		Credential: &syscall.Credential{
			Uid:    c.Spec.Process.User.UID,
			Gid:    c.Spec.Process.User.GID,
			Groups: c.Spec.Process.User.AdditionalGids,
		},
	}

	if c.Spec.Linux.UIDMappings != nil {
		var uidMappings []syscall.SysProcIDMap

		for _, uidMapping := range c.Spec.Linux.UIDMappings {
			uidMappings = append(uidMappings, syscall.SysProcIDMap{
				ContainerID: int(uidMapping.ContainerID),
				HostID:      int(uidMapping.HostID),
				Size:        int(uidMapping.Size),
			})
		}

		cmd.SysProcAttr.UidMappings = append(cmd.SysProcAttr.UidMappings, uidMappings...)
	}

	if c.Spec.Linux.GIDMappings != nil {
		var gidMappings []syscall.SysProcIDMap

		for _, gidMapping := range c.Spec.Linux.GIDMappings {
			gidMappings = append(gidMappings, syscall.SysProcIDMap{
				ContainerID: int(gidMapping.ContainerID),
				HostID:      int(gidMapping.HostID),
				Size:        int(gidMapping.Size),
			})
		}

		cmd.SysProcAttr.GidMappings = append(cmd.SysProcAttr.GidMappings, gidMappings...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := c.ExecHooks("startContainer", log); err != nil {
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	// we can't get logs or anything past this point
	// syscall.Seteuid(int(c.Spec.Process.User.UID))
	// syscall.Setegid(int(c.Spec.Process.User.GID))
	log.Info().Str("cmd", cmd.Args[0]).Any("args", cmd.Args[1:]).Msg("process")
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("(run) error executing in reexec2")
		return err
	}

	return nil
}
