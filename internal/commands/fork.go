package commands

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/nixpig/brownie/internal/capabilities"
	"github.com/nixpig/brownie/internal/cgroups"
	"github.com/nixpig/brownie/internal/container"
	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/nixpig/brownie/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

type ForkOpts struct {
	ID                string
	InitSockAddr      string
	ConsoleSocketFD   int
	ConsoleSocketPath string
}

func Fork(opts *ForkOpts, log *zerolog.Logger) error {
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		log.Error().Err(err).Msg("load existing container for fork")
		return err
	}

	initCh, initCloser, err := ipc.NewSender(opts.InitSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("new init ipc sender")
		return err
	}
	defer initCloser()

	if opts.ConsoleSocketFD != 0 {
		pty, err := terminal.NewPty()
		if err != nil {
			log.Error().Err(err).Msg("create new pty")
			return err
		}
		defer pty.Close()

		if err := pty.Connect(); err != nil {
			log.Error().Err(err).Msg("connect to pty")
			return err
		}

		consoleSocketPty := terminal.OpenPtySocket(
			opts.ConsoleSocketFD,
			opts.ConsoleSocketPath,
		)
		defer consoleSocketPty.Close()

		// FIXME: how do we pass ptysocket struct between fork?
		if err := consoleSocketPty.SendMsg(pty); err != nil {
			log.Error().Err(err).Msg("send message to pty")
			return err
		}
	}

	if err := filesystem.MountRootfs(cntr.Rootfs); err != nil {
		log.Error().Err(err).Msg("mount rootfs")
		return err
	}

	if err := filesystem.MountDevice(
		filesystem.Device{
			Source: "proc",
			Target: filepath.Join(cntr.Rootfs, "proc"),
			Fstype: "proc",
			Flags:  uintptr(0),
			Data:   "",
		},
	); err != nil {
		log.Error().Err(err).Msg("mount proc")
		return err
	}

	if err := filesystem.MountSpecMounts(
		cntr.Spec.Mounts,
		cntr.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("mount spec mounts")
	}

	if err := filesystem.MountDevices(
		filesystem.DefaultDevices,
		cntr.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("mount default devices")
		return err
	}

	for _, dev := range cntr.Spec.Linux.Devices {
		var absPath string
		if strings.Index(dev.Path, "/") == 0 {
			relPath := strings.TrimPrefix(dev.Path, "/")
			absPath = filepath.Join(cntr.Rootfs, relPath)
		} else {
			absPath = filepath.Join(cntr.Rootfs, dev.Path)
		}

		if err := unix.Mknod(
			absPath,
			uint32(*dev.FileMode),
			int(unix.Mkdev(uint32(dev.Major), uint32(dev.Minor))),
		); err != nil {
			log.Error().Err(err).
				Str("path", absPath).
				Uint32("fileMode", uint32(*dev.FileMode)).
				Int64("major", dev.Major).
				Int64("minor", dev.Minor).
				Any("dev", dev).Msg("make device node")
			return err
		}

		if err := os.Chown(
			absPath,
			int(*dev.UID),
			int(*dev.GID),
		); err != nil {
			log.Error().Err(err).
				Str("path", absPath).
				Uint32("UID", *dev.UID).
				Uint32("GID", *dev.GID).
				Msg("chown device node")
			return err
		}
	}

	if err := filesystem.CreateSymlinks(
		filesystem.DefaultSymlinks,
		cntr.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("create default symlinks")
		return err
	}

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(cntr.SockAddr); err != nil {
		log.Error().Err(err).Msg("remove existing container socket")
		return err
	}
	listener, err := net.Listen("unix", cntr.SockAddr)
	if err != nil {
		log.Error().Err(err).Msg("listen on container socket")
		return err
	}
	defer listener.Close()

	// FIXME: this isn't the correct PID - it should be from _inside_ container, 0 ??
	cntr.State.Pid = os.Getpid()
	log.Info().Int("pid", cntr.State.Pid).Msg("setting pid")
	if err := cntr.State.Save(); err != nil {
		log.Error().Err(err).Msg("save state after updating pid")
		return err
	}

	if err := filesystem.PivotRootfs(cntr.Rootfs); err != nil {
		log.Error().Err(err).Msg("pivot rootfs")
		return err
	}

	if slices.ContainsFunc(
		cntr.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	) {
		if err := syscall.Sethostname(
			[]byte(cntr.Spec.Hostname),
		); err != nil {
			log.Error().Err(err).
				Str("hostname", cntr.Spec.Hostname).
				Msg("set hostname")
			return err
		}

		if err := syscall.Setdomainname(
			[]byte(cntr.Spec.Domainname),
		); err != nil {
			log.Error().Err(err).
				Str("domainname", cntr.Spec.Domainname).
				Msg("set domainname")
			return err
		}
	}

	if err := capabilities.SetCapabilities(
		cntr.Spec.Process.Capabilities,
	); err != nil {
		log.Error().Err(err).Msg("set capabilities")
		return err
	}

	if err := cgroups.SetRlimits(cntr.Spec.Process.Rlimits); err != nil {
		log.Error().Err(err).Msg("set rlimits")
		return err
	}

	initCh <- []byte("ready")

	containerConn, err := listener.Accept()
	if err != nil {
		log.Error().Err(err).Msg("accept connection")
		return err
	}
	defer containerConn.Close()

	return listen(containerConn, cntr, log)
}

func listen(
	conn net.Conn,
	cntr *container.Container,
	log *zerolog.Logger,
) error {
	b := make([]byte, 128)

	n, err := conn.Read(b)
	if err != nil {
		log.Error().Err(err).Msg("read from connection")
		return err
	}

	if n == 0 {
		return errors.New("read zero bytes")
	}

	switch string(b[:n]) {
	case "start":
		cmd := exec.Command(
			cntr.Spec.Process.Args[0],
			cntr.Spec.Process.Args[1:]...,
		)

		cmd.Dir = cntr.Spec.Process.Cwd

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}

	return nil
}
