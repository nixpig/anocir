package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/nixpig/brownie/capabilities"
	"github.com/nixpig/brownie/cgroups"
	"github.com/nixpig/brownie/filesystem"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/nixpig/brownie/user"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Reexec(log *zerolog.Logger) error {
	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewSender(filepath.Join(containerRootDir, c.ID(), initSockFilename))
	if err != nil {
		return fmt.Errorf("create init sock sender: %w", err)
	}
	defer c.initIPC.closer()

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(
		filepath.Join(containerRootDir, c.ID(), containerSockFilename),
	); err != nil {
		return fmt.Errorf("remove socket before creating: %w", err)
	}

	listCh, listCloser, err := ipc.NewReceiver(filepath.Join(containerRootDir, c.ID(), containerSockFilename))
	if err != nil {
		return fmt.Errorf("create new socket receiver channel: %w", err)
	}
	defer listCloser()

	if err := filesystem.SetupRootfs(c.Rootfs(), c.Spec); err != nil {
		return fmt.Errorf("setup rootfs: %w", err)
	}

	c.initIPC.ch <- []byte("ready")

	if err := ipc.WaitForMsg(listCh, "start", func() error {
		if c.Spec.Process == nil {
			return errors.New("process is required")
		}

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
			if err := syscall.Sethostname([]byte(c.Spec.Hostname)); err != nil {
				return err
			}

			if err := syscall.Setdomainname([]byte(c.Spec.Domainname)); err != nil {
				return err
			}
		}

		if c.Spec.Process.Rlimits != nil {
			if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
				return err
			}
		}

		if c.Spec.Process.Capabilities != nil {
			if err := capabilities.SetCapabilities(
				c.Spec.Process.Capabilities,
			); err != nil {
				return err
			}
		}

		cmd := exec.Command(
			c.Spec.Process.Args[0],
			c.Spec.Process.Args[1:]...,
		)

		cmd.Dir = c.Spec.Process.Cwd

		var ambientCapsFlags []uintptr
		if c.Spec.Process.Capabilities != nil {
			for _, cap := range c.Spec.Process.Capabilities.Ambient {
				ambientCapsFlags = append(
					ambientCapsFlags,
					uintptr(capabilities.Capabilities[cap]),
				)
			}
		}

		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags:   uintptr(0),
			Unshareflags: uintptr(0),
			AmbientCaps:  ambientCapsFlags,
			Credential: &syscall.Credential{
				Uid:    c.Spec.Process.User.UID,
				Gid:    c.Spec.Process.User.GID,
				Groups: c.Spec.Process.User.AdditionalGids,
			},
		}

		if c.Spec.Linux.UIDMappings != nil {
			cmd.SysProcAttr.UidMappings = user.BuildUIDMappings(c.Spec.Linux.UIDMappings)
		}

		if c.Spec.Linux.GIDMappings != nil {
			cmd.SysProcAttr.GidMappings = user.BuildGIDMappings(c.Spec.Linux.GIDMappings)
		}

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := c.ExecHooks("startContainer"); err != nil {
			return fmt.Errorf("execute startContainer hooks: %w", err)
		}

		// we can't get logs or anything past this point
		if err := cmd.Run(); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
