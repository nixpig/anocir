package commands

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/internal/capabilities"
	"github.com/nixpig/brownie/internal/cgroups"
	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"github.com/syndtr/gocapability/capability"
	"golang.org/x/sys/unix"
)

func server(conn net.Conn, containerID string, spec specs.Spec, log *zerolog.Logger) {
	defer conn.Close()

	b := make([]byte, 128)

	for {
		n, err := conn.Read(b)
		if err != nil {
			log.Error().Err(err).Msg("read from connection")
		}

		if n == 0 {
			break
		}

		switch string(b[:n]) {
		case "start":
			cmd := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)
			cmd.Dir = spec.Process.Cwd

			cmd.Stdin = nullReader{}
			cmd.Stdout = conn
			cmd.Stderr = conn

			var ambientCapsFlags []uintptr
			for _, cap := range spec.Process.Capabilities.Ambient {
				ambientCapsFlags = append(ambientCapsFlags, uintptr(capabilities.Capabilities[cap]))
			}

			cmd.SysProcAttr = &syscall.SysProcAttr{
				AmbientCaps: ambientCapsFlags,
			}

			if err := cmd.Run(); err != nil {
				log.Error().Err(err).Msg("error executing command")
				conn.Write([]byte(err.Error()))
			}

			return
		}
	}
}

type ForkStage string

var (
	ForkIntermediate ForkStage = "intermediate"
	ForkDetached     ForkStage = "detached"
)

type ForkOpts struct {
	ID                string
	InitSockAddr      string
	ContainerSockAddr string
	PID               int
	ConsoleSocketFD   int
	ConsoleSocketPath string
}

func Fork(opts *ForkOpts, log *zerolog.Logger) error {
	container, err := internal.LoadContainer(opts.ID)
	if err != nil {
		return fmt.Errorf("load existing container for fork: %w", err)
	}

	initConn, err := net.Dial("unix", opts.InitSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("dialing init socket")
		return err
	}
	defer initConn.Close()
	if err := os.RemoveAll(opts.ContainerSockAddr); err != nil {
		log.Error().Err(err).Msg("remove existing container socket")
		return err
	}

	listener, err := net.Listen("unix", opts.ContainerSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("listen on container socket")
		return err
	}
	defer listener.Close()

	if opts.ConsoleSocketFD != 0 {
		pty, err := terminal.NewPty()
		if err != nil {
			log.Error().Err(err).Msg("new pty")
			return err
		}
		defer pty.Close()

		if err := pty.Connect(); err != nil {
			log.Error().Err(err).Msg("connect pty")
			return err
		}

		consoleSocketPty := terminal.OpenPtySocket(
			opts.ConsoleSocketFD,
			opts.ConsoleSocketPath,
		)
		defer consoleSocketPty.Close()

		// how does ptysocket struct pass between fork?
		if err := consoleSocketPty.SendMsg(pty); err != nil {
			log.Error().Err(err).Msg("send pty msg")
		}
	}

	if err := filesystem.MountRootfs(container.Rootfs); err != nil {
		log.Error().Err(err).Msg("mount rootfs")
		return err
	}

	if err := filesystem.MountProc(container.Rootfs); err != nil {
		log.Error().Err(err).Msg("mount proc")
		return err
	}

	if err := filesystem.MountMounts(
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
			log.Error().Err(err).Any("dev", dev).Msg("make device node")
			return err
		}

		if err := os.Chown(absPath, int(*dev.UID), int(*dev.GID)); err != nil {
			log.Error().Err(err).Str("path", absPath).Uint32("UID", *dev.UID).Uint32("GID", *dev.GID).Msg("chown device node")
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

	if err := syscall.Sethostname([]byte(container.Spec.Hostname)); err != nil {
		log.Error().Err(err).Str("hostname", container.Spec.Hostname).Msg("set hostname")
		return err
	}

	if err := filesystem.PivotRootfs(container.Rootfs); err != nil {
		log.Error().Err(err).Msg("pivot rootfs")
		return err
	}

	if container.Spec.Process.Capabilities != nil {
		caps := container.Spec.Process.Capabilities

		c, err := capability.NewPid2(0)
		if err != nil {
			log.Error().Err(err).Msg("new capabilities")
		}

		c.Clear(capability.BOUNDING)
		c.Clear(capability.EFFECTIVE)
		c.Clear(capability.INHERITABLE)
		c.Clear(capability.PERMITTED)
		c.Clear(capability.AMBIENT)

		if caps.Ambient != nil {
			for _, e := range caps.Ambient {
				if v, ok := capabilities.Capabilities[e]; ok {
					c.Set(capability.AMBIENT, capability.Cap(v))
				} else {
					log.Error().Err(errors.New(fmt.Sprintf("set ambient capability: %s", e))).Msg("set ambient capability")
					continue
				}
			}
		}

		if caps.Bounding != nil {
			for _, e := range caps.Bounding {
				if v, ok := capabilities.Capabilities[e]; ok {
					c.Set(capability.BOUNDING, capability.Cap(v))
				} else {
					log.Error().Err(errors.New(fmt.Sprintf("set bounding capability: %s", e))).Msg("set bounding capability")
					continue
				}
			}
		}

		if caps.Effective != nil {
			for _, e := range caps.Effective {
				if v, ok := capabilities.Capabilities[e]; ok {
					c.Set(capability.EFFECTIVE, capability.Cap(v))
				} else {
					log.Error().Err(errors.New(fmt.Sprintf("set effective capability: %s", e))).Msg("set effective capability")
					continue
				}
			}
		}

		if caps.Permitted != nil {
			for _, e := range caps.Permitted {
				if v, ok := capabilities.Capabilities[e]; ok {
					c.Set(capability.PERMITTED, capability.Cap(v))
				} else {
					log.Error().Err(errors.New(fmt.Sprintf("set permitted capability: %s", e))).Msg("set permitted capability")
					continue
				}
			}
		}

		if caps.Inheritable != nil {
			for _, e := range caps.Inheritable {
				if v, ok := capabilities.Capabilities[e]; ok {
					c.Set(capability.INHERITABLE, capability.Cap(v))
				} else {
					log.Error().Err(errors.New(fmt.Sprintf("set inheritable capability: %s", e))).Msg("set inheritable capability")
				}
			}
		}

		if err := c.Apply(
			capability.INHERITABLE |
				capability.EFFECTIVE |
				capability.BOUNDING |
				capability.PERMITTED |
				capability.AMBIENT,
		); err != nil {
			log.Error().Err(err).Msg("set capabilities")
		}
	}

	for _, rl := range container.Spec.Process.Rlimits {
		if err := syscall.Setrlimit(int(cgroups.Rlimits[rl.Type]), &syscall.Rlimit{
			Cur: rl.Soft,
			Max: rl.Hard,
		}); err != nil {
			log.Error().Err(err).Str("type", rl.Type).Msg("set rlimit")
		}
	}

	if n, err := initConn.Write([]byte("ready")); n == 0 || err != nil {
		log.Error().Err(err).Msg("send 'ready' message")
		return err
	}

	initConn.Close()

	for {
		containerConn, err := listener.Accept()
		if err != nil {
			log.Error().Err(err).Msg("accept connection")
			continue
		}

		go server(containerConn, opts.ID, *container.Spec, log)
	}
}

type nullReader struct{}

func (nullReader) Read(p []byte) (n int, err error) { return len(p), nil }
