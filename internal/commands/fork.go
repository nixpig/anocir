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
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

const containerSockFilename = "container.sock"

type ForkOpts struct {
	ID                string
	InitSockAddr      string
	ConsoleSocketFD   int
	ConsoleSocketPath string
}

func Fork(opts *ForkOpts, log *zerolog.Logger) error {
	root := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)

	cntr, err := container.Load(root)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return err
	}

	initCh, initCloser, err := ipc.NewSender(opts.InitSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("failed to create init ipc sender")
		return err
	}
	defer initCloser()

	if opts.ConsoleSocketFD != 0 {
		pty, err := terminal.NewPty()
		if err != nil {
			log.Error().Err(err).Msg("failed to create new terminal pty")
			return err
		}
		defer pty.Close()

		if err := pty.Connect(); err != nil {
			log.Error().Err(err).Msg("failed to connect to pty")
			return err
		}

		consoleSocketPty := terminal.OpenPtySocket(
			opts.ConsoleSocketFD,
			opts.ConsoleSocketPath,
		)
		defer consoleSocketPty.Close()

		// FIXME: how do we pass ptysocket struct between fork?
		if err := consoleSocketPty.SendMsg(pty); err != nil {
			log.Error().Err(err).Msg("failed to send message on terminal pty socket")
			return err
		}
	}

	rootfs := "rootfs"
	if cntr.Spec.Root != nil {
		rootfs = cntr.Spec.Root.Path
	}

	rootfs = filepath.Join(cntr.Root, rootfs)

	if err := filesystem.MountRootfs(rootfs); err != nil {
		log.Error().Err(err).Msg("failed to mount container rootfs")
		return err
	}

	if err := filesystem.MountDevice(
		filesystem.Device{
			Source: "proc",
			Target: filepath.Join(rootfs, "proc"),
			Fstype: "proc",
			Flags:  uintptr(0),
			Data:   "",
		},
	); err != nil {
		log.Error().Err(err).Msg("failed to mount proc device")
		return err
	}

	if err := filesystem.MountSpecMounts(
		cntr.Spec.Mounts,
		rootfs,
	); err != nil {
		log.Error().Err(err).Msg("failed to mount spec mount points")
	}

	if err := filesystem.MountDevices(
		filesystem.DefaultDevices,
		rootfs,
	); err != nil {
		log.Error().Err(err).Msg("failed to mount default devices")
		return err
	}

	for _, dev := range cntr.Spec.Linux.Devices {
		var absPath string
		if strings.Index(dev.Path, "/") == 0 {
			relPath := strings.TrimPrefix(dev.Path, "/")
			absPath = filepath.Join(rootfs, relPath)
		} else {
			absPath = filepath.Join(rootfs, dev.Path)
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
		rootfs,
	); err != nil {
		log.Error().Err(err).Msg("failed to create default symlinks")
		return err
	}

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(
		filepath.Join(cntr.Root, containerSockFilename),
	); err != nil {
		log.Error().Err(err).Msg("failed to remove existing container socket")
		return err
	}

	listener, err := net.Listen(
		"unix",
		filepath.Join(cntr.Root, containerSockFilename),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to listen on container socket")
		return err
	}
	defer listener.Close()

	if err := filesystem.PivotRootfs(rootfs); err != nil {
		log.Error().Err(err).Msg("failed to pivot container rootfs")
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
				Msg("failed to set hostname")
			return err
		}

		if err := syscall.Setdomainname(
			[]byte(cntr.Spec.Domainname),
		); err != nil {
			log.Error().Err(err).
				Str("domainname", cntr.Spec.Domainname).
				Msg("faile to set domainname")
			return err
		}
	}

	if err := capabilities.SetCapabilities(
		cntr.Spec.Process.Capabilities,
	); err != nil {
		log.Error().Err(err).Msg("failed to set capabilities")
		return err
	}

	if err := cgroups.SetRlimits(cntr.Spec.Process.Rlimits); err != nil {
		log.Error().Err(err).Msg("failed to 1set rlimits")
		return err
	}

	initCh <- []byte("ready")

	containerConn, err := listener.Accept()
	if err != nil {
		log.Error().Err(err).Msg("failed to accept on container connection")
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
			cmd := exec.Command(
				cntr.Spec.Process.Args[0],
				cntr.Spec.Process.Args[1:]...,
			)

			cmd.Dir = cntr.Spec.Process.Cwd

			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				log.Error().Err(err).Msg("failed to start process in container")
			}

		}
	}
}
