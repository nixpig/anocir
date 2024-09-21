package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nixpig/brownie/internal/capabilities"
	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/pkg"
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
				log.Error().Err(err).Msg("error executing underlying command")
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
	ConsoleSocket     string
}

func Fork(opts *ForkOpts, log *zerolog.Logger) error {
	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)
	specJSON, err := os.ReadFile(filepath.Join(containerPath, "config.json"))
	if err != nil {
		log.Error().Err(err).Msg("read spec")
		return err
	}

	var spec specs.Spec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		log.Error().Err(err).Msg("parse config")
		return err
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

	containerRootfs := filepath.Join(containerPath, spec.Root.Path)

	if err := filesystem.MountProc(containerRootfs); err != nil {
		initConn.Write([]byte(fmt.Sprintf("mount proc: %s", err)))
	}

	if err := filesystem.MountRootfs(containerRootfs); err != nil {
		log.Error().Err(err).Msg("mount rootfs")
		return err
	}

	for _, mount := range spec.Mounts {
		// if mount.Destination == "/dev" {
		// 	continue
		// }
		var dest string
		if strings.Index(mount.Destination, "/") == 0 {
			dest = containerRootfs + mount.Destination
		} else {
			dest = mount.Destination
		}

		if _, err := os.Stat(dest); err != nil {
			if !os.IsNotExist(err) {
				log.Error().Err(err).Msg("check if spec mount destination exists")
				return err
			}

			if err := os.MkdirAll(dest, os.ModeDir); err != nil {
				log.Error().Err(err).Str("dest", dest).Msg("make spec mount dir")
				return err
			}
		}

		var flags uintptr
		if mount.Type == "bind" {
			flags = flags | syscall.MS_BIND
		}

		var dataOptions []string
		for _, opt := range mount.Options {
			o, ok := pkg.MountOptions[opt]
			if !ok {
				if !strings.HasPrefix(opt, "gid=") &&
					!strings.HasPrefix(opt, "uid=") &&
					opt != "newinstance" {
					dataOptions = append(dataOptions, opt)
				}
			} else {
				if !o.No {
					flags = flags | o.Flag
				}
			}
		}

		var data string
		if len(dataOptions) > 0 {
			data = strings.Join(dataOptions, ",")
		}

		if err := syscall.Mount(
			mount.Source,
			dest,
			mount.Type,
			flags,
			data,
		); err != nil {
			log.Error().Err(err).
				Any("mount", mount).
				Msg("mount spec mount")
			return err
		}
	}

	for _, dev := range filesystem.DefaultDevices {
		var absPath string
		if strings.Index(dev.Path, "/") == 0 {
			relPath := strings.TrimPrefix(dev.Path, "/")
			absPath = filepath.Join(containerRootfs, relPath)
		} else {
			absPath = filepath.Join(containerRootfs, dev.Path)
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			f, err := os.Create(absPath)
			if err != nil && !os.IsExist(err) {
				log.Error().Err(err).Msg("create default device mount point")
				return err
			}
			if f != nil {
				f.Close()
			}
			if err := syscall.Mount(
				dev.Path,
				absPath,
				"bind",
				syscall.MS_BIND,
				"",
			); err != nil {
				log.Error().Err(err).Msg("bind mount default devices")
				return err
			}
		}
	}

	for _, dev := range spec.Linux.Devices {
		var absPath string
		if strings.Index(dev.Path, "/") == 0 {
			relPath := strings.TrimPrefix(dev.Path, "/")
			absPath = filepath.Join(containerRootfs, relPath)
		} else {
			absPath = filepath.Join(containerRootfs, dev.Path)
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

	for oldname, newname := range filesystem.DefaultSymlinks {
		nn := filepath.Join(containerRootfs, newname)
		if err := os.Symlink(oldname, nn); err != nil {
			log.Error().Err(err).Str("newname", newname).Str("oldname", oldname).Msg("link default file descriptors")
			return err
		}
	}

	if opts.ConsoleSocket != "" {
		f, err := os.Create(filepath.Join(containerRootfs, "dev/console"))
		if err != nil && !os.IsExist(err) {
			log.Error().Err(err).Msg("create /dev/console")
			return err
		}
		if f != nil {
			if err := f.Chmod(0666); err != nil {
				log.Error().Err(err).Msg("set permission of /dev/console")
				return err
			}
			f.Close()
		}
		if err := syscall.Mount(
			filepath.Join(containerRootfs, "dev/console"),
			opts.ConsoleSocket,
			"bind",
			syscall.MS_BIND,
			"",
		); err != nil {
			log.Error().Err(err).Msg("bind mount /dev/console")
			return err
		}
	}

	if err := syscall.Sethostname([]byte(spec.Hostname)); err != nil {
		log.Error().Err(err).Str("hostname", spec.Hostname).Msg("set hostname")
		return err
	}

	if err := filesystem.PivotRootfs(containerRootfs); err != nil {
		log.Error().Err(err).Msg("pivot rootfs")
		return err
	}

	if spec.Process.Capabilities != nil {
		caps := spec.Process.Capabilities

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

	for _, rl := range spec.Process.Rlimits {
		if err := syscall.Setrlimit(int(pkg.Rlimits[rl.Type]), &syscall.Rlimit{
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

		go server(containerConn, opts.ID, spec, log)
	}
}

type nullReader struct{}

func (nullReader) Read(p []byte) (n int, err error) { return len(p), nil }
