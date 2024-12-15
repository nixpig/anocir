package container

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/nixpig/brownie/capabilities"
	"github.com/nixpig/brownie/cgroups"
	"github.com/nixpig/brownie/filesystem"
	"github.com/nixpig/brownie/user"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Reexec() error {
	if err := filesystem.SetupRootfs(c.Rootfs(), c.Spec); err != nil {
		return fmt.Errorf("setup rootfs: %w", err)
	}

	// send "ready"
	initConn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, c.ID(), initSockFilename),
	)
	if err != nil {
		return err
	}

	initConn.Write([]byte("ready"))
	// close asap so it doesn't leak into the container
	initConn.Close()

	// wait for "start"
	if err := os.RemoveAll(
		filepath.Join(containerRootDir, c.ID(), containerSockFilename),
	); err != nil {
		return fmt.Errorf("remove any existing container socket: %w", err)
	}

	listener, err := net.Listen(
		"unix",
		filepath.Join(containerRootDir, c.ID(), containerSockFilename),
	)
	if err != nil {
		return err
	}
	containerConn, err := listener.Accept()
	if err != nil {
		return err
	}

	b := make([]byte, 128)
	n, err := containerConn.Read(b)
	if err != nil {
		return fmt.Errorf("read from container socket: %w", err)
	}

	msg := string(b[:n])
	if msg != "start" {
		return fmt.Errorf("expecting 'start', received '%s'", msg)
	}

	// close as soon as we're done so they don't leak into the container
	containerConn.Close()
	listener.Close()

	// after receiving "start"
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

	if err := filesystem.SetRootfsMountPropagation(
		c.Spec.Linux.RootfsPropagation,
	); err != nil {
		return err
	}

	if err := filesystem.MountRootReadonly(
		c.Spec.Root.Readonly,
	); err != nil {
		return err
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

	if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
		return err
	}

	if err := capabilities.SetCapabilities(
		c.Spec.Process.Capabilities,
	); err != nil {
		return err
	}

	cmd := exec.Command(c.Spec.Process.Args[0], c.Spec.Process.Args[1:]...)

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
		cmd.SysProcAttr.UidMappings =
			user.BuildUIDMappings(c.Spec.Linux.UIDMappings)
	}

	if c.Spec.Linux.GIDMappings != nil {
		cmd.SysProcAttr.GidMappings =
			user.BuildGIDMappings(c.Spec.Linux.GIDMappings)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := c.ExecHooks("startContainer"); err != nil {
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	// point of no return
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
