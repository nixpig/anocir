package commands

import (
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
	container, err := container.LoadContainer(opts.ID)
	if err != nil {
		log.Error().Err(err).Msg("load existing container for fork")
		return err
	}

	initCh, _, err := ipc.NewSender(opts.InitSockAddr)
	if err != nil {
		log.Error().Err(err).
			Str("InitSockAddr", opts.InitSockAddr).
			Msg("create ipc sender for init channel")
	}

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

	if err := filesystem.MountRootfs(container.Rootfs); err != nil {
		log.Error().Err(err).Msg("mount rootfs")
		return err
	}

	if err := filesystem.MountDevice(
		filesystem.Device{
			Source: "proc",
			Target: filepath.Join(container.Rootfs, "proc"),
			Fstype: "proc",
			Flags:  uintptr(0),
			Data:   "",
		},
	); err != nil {
		log.Error().Err(err).Msg("mount proc")
		return err
	}

	if err := filesystem.MountSpecMounts(
		container.Spec.Mounts,
		container.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("mount spec mounts")
	}

	if err := filesystem.MountDevices(
		filesystem.DefaultDevices,
		container.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("mount default devices")
		return err
	}

	for _, dev := range container.Spec.Linux.Devices {
		var absPath string
		if strings.Index(dev.Path, "/") == 0 {
			relPath := strings.TrimPrefix(dev.Path, "/")
			absPath = filepath.Join(container.Rootfs, relPath)
		} else {
			absPath = filepath.Join(container.Rootfs, dev.Path)
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
		container.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("create default symlinks")
		return err
	}

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(container.SockAddr); err != nil {
		log.Error().Err(err).Msg("remove existing container socket")
		return err
	}
	containerCh, containerCloser, err := ipc.NewReceiver(container.SockAddr)
	if err != nil {
		log.Error().Err(err).Msg("new container ipc receiver")
		return err
	}
	defer containerCloser()

	// FIXME: this isn't the correct PID - it should be from _inside_ container, 0 ??
	container.State.Pid = os.Getpid()
	log.Info().Int("pid", container.State.Pid).Msg("setting pid")
	if err := container.State.Save(); err != nil {
		log.Error().Err(err).Msg("save state after updating pid")
		return err
	}

	if err := filesystem.PivotRootfs(container.Rootfs); err != nil {
		log.Error().Err(err).Msg("pivot rootfs")
		return err
	}

	if slices.ContainsFunc(
		container.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	) {
		if err := syscall.Sethostname(
			[]byte(container.Spec.Hostname),
		); err != nil {
			log.Error().Err(err).
				Str("hostname", container.Spec.Hostname).
				Msg("set hostname")
			return err
		}

		if err := syscall.Setdomainname(
			[]byte(container.Spec.Domainname),
		); err != nil {
			log.Error().Err(err).
				Str("domainname", container.Spec.Domainname).
				Msg("set domainname")
			return err
		}
	}

	if err := capabilities.SetCapabilities(
		container.Spec.Process.Capabilities,
	); err != nil {
		log.Error().Err(err).Msg("set capabilities")
		return err
	}

	if err := cgroups.SetRlimits(container.Spec.Process.Rlimits); err != nil {
		log.Error().Err(err).Msg("set rlimits")
		return err
	}

	// notify host that container is ready
	initCh <- []byte("ready")

	return waitForMsg(containerCh, container, log)
}

func waitForMsg(
	ch chan []byte,
	container *container.Container,
	log *zerolog.Logger,
) error {
	for {
		msg := <-ch

		log.Info().
			Str("msg", string(msg)).
			Msg("container received ipc msg")

		switch string(msg) {
		case "start":
			cmd := exec.Command(
				container.Spec.Process.Args[0],
				container.Spec.Process.Args[1:]...,
			)

			cmd.Dir = container.Spec.Process.Cwd

			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			return cmd.Run()

		default:
			log.Warn().
				Str("msg", string(msg)).
				Msg("unknown command")
		}
	}
}
