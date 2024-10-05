package commands

import (
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
	log.Info().Any("opts", opts).Msg("run fork command")
	log.Info().Str("id", opts.ID).Msg("load container")
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return err
	}

	log.Info().Msg("create init ipc sender")
	initCh, initCloser, err := ipc.NewSender(opts.InitSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("failed to create init ipc sender")
		return err
	}
	defer initCloser()

	if opts.ConsoleSocketFD != 0 {
		log.Info().Msg("create new terminal pty")
		pty, err := terminal.NewPty()
		if err != nil {
			log.Error().Err(err).Msg("failed to create new terminal pty")
			return err
		}
		defer pty.Close()

		log.Info().Msg("connect to terminal pty")
		if err := pty.Connect(); err != nil {
			log.Error().Err(err).Msg("failed to connect to pty")
			return err
		}

		log.Info().Msg("open terminal pty socket")
		consoleSocketPty := terminal.OpenPtySocket(
			opts.ConsoleSocketFD,
			opts.ConsoleSocketPath,
		)
		defer consoleSocketPty.Close()

		// FIXME: how do we pass ptysocket struct between fork?
		log.Info().Msg("send message on terminal pty socket")
		if err := consoleSocketPty.SendMsg(pty); err != nil {
			log.Error().Err(err).Msg("failed to send message on terminal pty socket")
			return err
		}
	}

	log.Info().Str("rootfs", cntr.Rootfs).Msg("mount container rootfs")
	if err := filesystem.MountRootfs(cntr.Rootfs); err != nil {
		log.Error().Err(err).Msg("failed to mount container rootfs")
		return err
	}

	log.Info().Msg("mount proc device")
	if err := filesystem.MountDevice(
		filesystem.Device{
			Source: "proc",
			Target: filepath.Join(cntr.Rootfs, "proc"),
			Fstype: "proc",
			Flags:  uintptr(0),
			Data:   "",
		},
	); err != nil {
		log.Error().Err(err).Msg("failed to mount proc device")
		return err
	}

	log.Info().Msg("mount spec mount points")
	if err := filesystem.MountSpecMounts(
		cntr.Spec.Mounts,
		cntr.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("failed to mount spec mount points")
	}

	log.Info().Msg("mount default devices")
	if err := filesystem.MountDevices(
		filesystem.DefaultDevices,
		cntr.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("failed to mount default devices")
		return err
	}

	log.Info().Msg("mount spec devices")
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

	log.Info().Msg("create default symlinks")
	if err := filesystem.CreateSymlinks(
		filesystem.DefaultSymlinks,
		cntr.Rootfs,
	); err != nil {
		log.Error().Err(err).Msg("failed to create default symlinks")
		return err
	}

	// set up the socket _before_ pivot root
	log.Info().Str("sockaddr", cntr.SockAddr).Msg("remove existing container sockaddr")
	if err := os.RemoveAll(cntr.SockAddr); err != nil {
		log.Error().Err(err).Msg("failed to remove existing container socket")
		return err
	}

	log.Info().Str("sockaddr", cntr.SockAddr).Msg("listen on container socket")
	listener, err := net.Listen("unix", cntr.SockAddr)
	if err != nil {
		log.Error().Err(err).Msg("failed to listen on container socket")
		return err
	}
	defer listener.Close()

	// FIXME: this isn't the correct PID - it should be from _inside_ container, 0 ??
	cntr.State.Pid = os.Getpid()
	log.Info().Int("pid", cntr.State.Pid).Msg("get/set pid and save state")
	if err := cntr.State.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save state with pid")
		return err
	}

	log.Info().Str("rootfs", cntr.Rootfs).Msg("pivot container rootfs")
	if err := filesystem.PivotRootfs(cntr.Rootfs); err != nil {
		log.Error().Err(err).Msg("failed to pivot container rootfs")
		return err
	}

	if slices.ContainsFunc(
		cntr.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	) {
		log.Info().Str("hostname", cntr.Spec.Hostname).Msg("set hostname")
		if err := syscall.Sethostname(
			[]byte(cntr.Spec.Hostname),
		); err != nil {
			log.Error().Err(err).
				Str("hostname", cntr.Spec.Hostname).
				Msg("failed to set hostname")
			return err
		}

		log.Info().Str("domainname", cntr.Spec.Domainname).Msg("set domainname")
		if err := syscall.Setdomainname(
			[]byte(cntr.Spec.Domainname),
		); err != nil {
			log.Error().Err(err).
				Str("domainname", cntr.Spec.Domainname).
				Msg("faile to set domainname")
			return err
		}
	}

	log.Info().
		Any("capabilities", cntr.Spec.Process.Capabilities).
		Msg("set capabilities")
	if err := capabilities.SetCapabilities(
		cntr.Spec.Process.Capabilities,
	); err != nil {
		log.Error().Err(err).Msg("failed to set capabilities")
		return err
	}

	log.Info().Any("rlimits", cntr.Spec.Process.Rlimits).Msg("set rlimits")
	if err := cgroups.SetRlimits(cntr.Spec.Process.Rlimits); err != nil {
		log.Error().Err(err).Msg("failed to 1set rlimits")
		return err
	}

	log.Info().Msg("send 'ready' message to init channel")
	initCh <- []byte("ready")

	log.Info().Msg("accept on container connection")
	containerConn, err := listener.Accept()
	if err != nil {
		log.Error().Err(err).Msg("failed to accept on container connection")
		return err
	}
	defer containerConn.Close()

	log.Info().Msg("listen on container connection")
	return listen(containerConn, cntr, log)
}

func listen(
	conn net.Conn,
	cntr *container.Container,
	log *zerolog.Logger,
) error {
	log.Info().Msg("waiting for message in fork")
	for {
		b := make([]byte, 128)

		n, _ := conn.Read(b)
		// if err != nil {
		// 	log.Error().Err(err).Msg("read from connection")
		// 	return err
		// }
		//
		if n == 0 {
			continue
		}

		switch string(b[:n]) {
		case "start":
			log.Info().
				Str("cmd", cntr.Spec.Process.Args[0]).
				Any("args", cntr.Spec.Process.Args[1:]).
				Msg("create command")
			cmd := exec.Command(
				cntr.Spec.Process.Args[0],
				cntr.Spec.Process.Args[1:]...,
			)

			log.Info().
				Str("cwd", cntr.Spec.Process.Cwd).
				Msg("set current working directory")
			cmd.Dir = cntr.Spec.Process.Cwd

			log.Info().
				Str("stdin", os.Stdin.Name()).
				Str("stdout", os.Stdout.Name()).
				Str("stderr", os.Stderr.Name()).
				Msg("attach stdio")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			log.Info().Msg("start process in container")
			if err := cmd.Run(); err != nil {
				log.Error().Err(err).Msg("failed to start process in container")
			}

		}
	}
}
